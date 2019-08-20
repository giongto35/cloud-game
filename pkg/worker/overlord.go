package worker

import (
	"log"

	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/giongto35/cloud-game/pkg/webrtc"
	room2 "github.com/giongto35/cloud-game/pkg/worker/room"
	"github.com/gorilla/websocket"
)

// OverlordClient maintans connection to overlord
// We expect only one OverlordClient for each server
type OverlordClient struct {
	*cws.Client
}

// NewOverlordClient returns a client connecting to overlord for coordiation between different server
func NewOverlordClient(oc *websocket.Conn) *OverlordClient {
	if oc == nil {
		return nil
	}

	oClient := &OverlordClient{
		Client: cws.NewClient(oc),
	}
	return oClient
}

// RouteOverlord are all routes server received from overlord
func (h *Handler) RouteOverlord() {
	iceCandidates := map[string][]string{}
	oClient := h.oClient

	// Received from overlord the serverID
	oClient.Receive(
		"serverID",
		func(response cws.WSPacket) (request cws.WSPacket) {
			// Stick session with serverID got from overlord
			log.Println("Received serverID ", response.Data)
			h.serverID = response.Data

			return cws.EmptyPacket
		},
	)

	oClient.Receive(
		"initwebrtc",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received relay SDP of a browser from overlord")
			peerconnection := webrtc.NewWebRTC()
			localSession, err := peerconnection.StartClient(resp.Data, iceCandidates[resp.SessionID])
			//h.peerconnections[resp.SessionID] = peerconnection

			// Create new sessions when we have new peerconnection initialized
			session := &Session{
				peerconnection: peerconnection,
			}
			h.sessions[resp.SessionID] = session

			log.Println("Start peerconnection", resp.SessionID)
			if err != nil {
				log.Println("Error: Cannot create new webrtc session", err)
				return cws.EmptyPacket
			}

			return cws.WSPacket{
				ID:        "sdp",
				Data:      localSession,
				SessionID: resp.SessionID,
			}
		},
	)

	oClient.Receive(
		"start",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a start request from overlord")
			session, ok := h.sessions[resp.SessionID]
			log.Println("Find ", resp.SessionID, session, ok)

			peerconnection := session.peerconnection
			room := h.startGameHandler(resp.Data, resp.RoomID, resp.PlayerIndex, peerconnection)
			session.RoomID = room.ID
			// TODO: can data race
			h.rooms[room.ID] = room

			return cws.WSPacket{
				ID:     "start",
				RoomID: room.ID,
			}
		},
	)

	oClient.Receive(
		"quit",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a quit request from overlord")
			session, ok := h.sessions[resp.SessionID]
			log.Println("Find ", resp.SessionID, session, ok)

			room := h.getRoom(session.RoomID)
			// Defensive coding, check if the peerconnection is in room
			if room.IsPCInRoom(session.peerconnection) {
				h.detachPeerConn(session.peerconnection)
			}

			return cws.EmptyPacket
		},
	)

	oClient.Receive(
		"save",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a save game from overlord")
			log.Println("RoomID:", resp.RoomID)
			req.ID = "save"
			req.Data = "ok"
			if resp.RoomID != "" {
				room := h.getRoom(resp.RoomID)
				if room == nil {
					return
				}
				err := room.SaveGame()
				if err != nil {
					log.Println("[!] Cannot save game state: ", err)
					req.Data = "error"
				}
			} else {
				req.Data = "error"
			}

			return req
		})

	oClient.Receive(
		"load",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a load game from overlord")
			log.Println("Loading game state")
			req.ID = "load"
			req.Data = "ok"
			if resp.RoomID != "" {
				room := h.getRoom(resp.RoomID)
				err := room.LoadGame()
				if err != nil {
					log.Println("[!] Cannot load game state: ", err)
					req.Data = "error"
				}
			} else {
				req.Data = "error"
			}

			return req
		})

	oClient.Receive(
		"icecandidate",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a icecandidate from overlord: ", resp.Data)
			iceCandidates[resp.SessionID] = append(iceCandidates[resp.SessionID], resp.Data)

			return cws.EmptyPacket
		},
	)

	oClient.Receive(
		"terminateSession",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a terminate session ", resp.SessionID)
			session, ok := h.sessions[resp.SessionID]
			log.Println("Find ", session, ok)
			if ok {
				session.Close()
				delete(h.sessions, resp.SessionID)
				h.detachPeerConn(session.peerconnection)
			}

			return cws.EmptyPacket
		},
	)
}

func getServerIDOfRoom(oc *OverlordClient, roomID string) string {
	log.Println("Request overlord roomID ", roomID)
	packet := oc.SyncSend(
		cws.WSPacket{
			ID:   "getRoom",
			Data: roomID,
		},
	)
	log.Println("Received roomID from overlord ", packet.Data)

	return packet.Data
}

func (h *Handler) startGameHandler(gameName, roomID string, playerIndex int, peerconnection *webrtc.WebRTC) *room2.Room {
	log.Println("Starting game")
	// If we are connecting to overlord, request corresponding serverID based on roomID
	// TODO: check if roomID is in the current server
	room := h.getRoom(roomID)
	log.Println("Got Room from local ", room, " ID: ", roomID)
	// If room is not running
	if room == nil {
		// Create new room
		room = h.createNewRoom(gameName, roomID, playerIndex)
		// Wait for done signal from room
		go func() {
			<-room.Done
			h.detachRoom(room.ID)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Println("Is PC in room", room.IsPCInRoom(peerconnection))
	if !room.IsPCInRoom(peerconnection) {
		h.detachPeerConn(peerconnection)
		room.AddConnectionToRoom(peerconnection, playerIndex)
	}

	// Register room to overlord if we are connecting to overlord
	if room != nil && h.oClient != nil {
		h.oClient.Send(cws.WSPacket{
			ID:   "registerRoom",
			Data: roomID,
		}, nil)
	}

	return room
}
