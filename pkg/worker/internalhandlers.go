package worker

import (
	"encoding/json"
	"log"
	"strconv"

	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

func (h *Handler) handleServerId() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("[worker] new id: %s", resp.Data)
		h.serverID = resp.Data
		// unlock worker if it's locked
		h.w.lock.Unlock()
		return
	}
}

func (h *Handler) handleTerminateSession() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a terminate session ", resp.SessionID)
		session := h.getSession(resp.SessionID)
		if session != nil {
			session.Close()
			delete(h.sessions, resp.SessionID)
			h.detachPeerConn(session.peerconnection)
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}

		return cws.EmptyPacket
	}
}

func (h *Handler) handleInitWebrtc() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a request to createOffer from browser via coordinator")

		peerconnection := webrtc.NewWebRTC().WithConfig(
			webrtcConfig.Config{Encoder: h.cfg.Encoder, Webrtc: h.cfg.Webrtc},
		)
		var initPacket struct {
			IsMobile bool `json:"is_mobile"`
		}
		err := json.Unmarshal([]byte(resp.Data), &initPacket)
		if err != nil {
			log.Println("Error: Cannot decode json:", err)
			return cws.EmptyPacket
		}

		localSession, err := peerconnection.StartClient(
			initPacket.IsMobile,
			// send back candidate string to browser
			func(cd string) { h.oClient.Send(api.IceCandidatePacket(cd, resp.SessionID), nil) },
		)

		// localSession, err := peerconnection.StartClient(initPacket.IsMobile, iceCandidates[resp.SessionID])
		// h.peerconnections[resp.SessionID] = peerconnection

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
			ID:   "offer",
			Data: localSession,
		}
	}
}

func (h *Handler) handleAnswer() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received answer SDP from browser")
		session := h.getSession(resp.SessionID)
		if session != nil {
			peerconnection := session.peerconnection

			err := peerconnection.SetRemoteSDP(resp.Data)
			if err != nil {
				log.Println("Error: Cannot set RemoteSDP of client: " + resp.SessionID)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}

func (h *Handler) handleIceCandidate() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received remote Ice Candidate from browser")
		session := h.getSession(resp.SessionID)

		if session != nil {
			peerconnection := session.peerconnection

			err := peerconnection.AddCandidate(resp.Data)
			if err != nil {
				log.Println("Error: Cannot add IceCandidate of client: " + resp.SessionID)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}

		return cws.EmptyPacket
	}
}

func (h *Handler) handleGameStart() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a start request from coordinator")
		session := h.getSession(resp.SessionID)
		if session == nil {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
			return cws.EmptyPacket
		}

		peerconnection := session.peerconnection
		// TODO: Standardize for all types of packet. Make WSPacket generic
		startPacket := api.GameStartCall{}
		if err := startPacket.From(resp.Data); err != nil {
			return cws.EmptyPacket
		}
		gameMeta := games.GameMetadata{
			Name: startPacket.Name,
			Type: startPacket.Type,
			Path: startPacket.Path,
		}

		room := h.startGameHandler(gameMeta, resp.RoomID, resp.PlayerIndex, peerconnection)
		session.RoomID = room.ID
		// TODO: can data race
		h.rooms[room.ID] = room

		return cws.WSPacket{ID: "start", RoomID: room.ID}
	}
}

func (h *Handler) handleGameQuit() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a quit request from coordinator")
		session := h.getSession(resp.SessionID)

		if session != nil {
			room := h.getRoom(session.RoomID)
			// Defensive coding, check if the peerconnection is in room
			if room.IsPCInRoom(session.peerconnection) {
				h.detachPeerConn(session.peerconnection)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}

		return cws.EmptyPacket
	}
}

func (h *Handler) handleGameSave() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a save game from coordinator")
		log.Println("RoomID:", resp.RoomID)
		req.ID = api.GameSave
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
	}
}

func (h *Handler) handleGameLoad() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a load game from coordinator")
		log.Println("Loading game state")
		req.ID = api.GameLoad
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
	}
}

func (h *Handler) handleGamePlayerSelect() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received an update player index event from coordinator")
		req.ID = api.GamePlayerSelect

		room := h.getRoom(resp.RoomID)
		session := h.getSession(resp.SessionID)
		idx, err := strconv.Atoi(resp.Data)
		log.Printf("Got session %v and room %v", session, room)

		if room != nil && session != nil && err == nil {
			room.UpdatePlayerIndex(session.peerconnection, idx)
			req.Data = strconv.Itoa(idx)
		} else {
			req.Data = "error"
		}

		return req
	}
}

func (h *Handler) handleGameMultitap() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a multitap toggle from coordinator")
		req.ID = api.GameMultitap
		req.Data = "ok"
		if resp.RoomID != "" {
			room := h.getRoom(resp.RoomID)
			err := room.ToggleMultitap()
			if err != nil {
				log.Println("[!] Could not toggle multitap state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}

		return req
	}
}

// startGameHandler starts a game if roomID is given, if not create new room
func (h *Handler) startGameHandler(game games.GameMetadata, existedRoomID string, playerIndex int, peerconnection *webrtc.WebRTC) *room.Room {
	log.Printf("Loading game: %v\n", game.Name)
	// If we are connecting to coordinator, request corresponding serverID based on roomID
	// TODO: check if existedRoomID is in the current server
	room := h.getRoom(existedRoomID)
	// If room is not running
	if room == nil {
		log.Println("Got Room from local ", room, " ID: ", existedRoomID)
		// Create new room and update player index
		room = h.createNewRoom(game, existedRoomID)
		room.UpdatePlayerIndex(peerconnection, playerIndex)

		// Wait for done signal from room
		go func() {
			<-room.Done
			h.detachRoom(room.ID)
			// send signal to coordinator that the room is closed, coordinator will remove that room
			h.oClient.Send(api.CloseRoomPacket(room.ID), nil)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Println("Is PC in room", room.IsPCInRoom(peerconnection))
	if !room.IsPCInRoom(peerconnection) {
		h.detachPeerConn(peerconnection)
		room.AddConnectionToRoom(peerconnection)
	}

	// Register room to coordinator if we are connecting to coordinator
	if room != nil && h.oClient != nil {
		h.oClient.Send(api.RegisterRoomPacket(room.ID), nil)
	}

	return room
}
