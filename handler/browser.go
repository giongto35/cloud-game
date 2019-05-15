package handler

import (
	"encoding/json"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"
)

type BrowserClient struct {
	*cws.Client
}

// RouteBrowser are all routes server received from browser
func (s *Session) RouteBrowser() {
	browserClient := s.BrowserClient

	browserClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	browserClient.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received user SDP")
		localSession, err := s.peerconnection.StartClient(resp.Data, config.Width, config.Height)
		if err != nil {
			if err != nil {
				log.Println("Error: Cannot create new webrtc session", err)
				return cws.EmptyPacket
			}
		}

		return cws.WSPacket{
			ID:        "sdp",
			Data:      localSession,
			SessionID: s.ID,
		}
	})

	// TODO: Add save and load
	browserClient.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Saving game state")
		req.ID = "save"
		req.Data = "ok"
		if s.RoomID != "" {
			room := s.handler.getRoom(s.RoomID)
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

	browserClient.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Loading game state")
		req.ID = "load"
		req.Data = "ok"
		if s.RoomID != "" {
			room := s.handler.getRoom(s.RoomID)
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

	browserClient.Receive("start", func(resp cws.WSPacket) (req cws.WSPacket) {
		s.GameName = resp.Data
		s.RoomID = resp.RoomID
		s.PlayerIndex = resp.PlayerIndex

		log.Println("Starting game")
		// If we are connecting to overlord, request corresponding serverID based on roomID
		if s.OverlordClient != nil {
			roomServerID := getServerIDOfRoom(s.OverlordClient, s.RoomID)
			log.Println("Server of RoomID ", s.RoomID, " is ", roomServerID, " while current server is ", s.ServerID)
			// If the target serverID is different from current serverID
			if roomServerID != "" && s.ServerID != roomServerID {
				// TODO: Re -register
				// Bridge Connection to the target serverID
				go s.bridgeConnection(roomServerID, s.GameName, s.RoomID, s.PlayerIndex)
				return
			}
		}

		// Get Room in local server
		// TODO: check if roomID is in the current server
		room := s.handler.getRoom(s.RoomID)
		log.Println("Got Room from local ", room, " ID: ", s.RoomID)
		// If room is not running
		if room == nil {
			// Create new room
			room = s.handler.createNewRoom(s.GameName, s.RoomID, s.PlayerIndex)
		}

		// Attach peerconnection to room. If PC is already in room, don't detach
		log.Println("Is PC in room", room.IsPCInRoom(s.peerconnection))
		if !room.IsPCInRoom(s.peerconnection) {
			s.handler.detachPeerConn(s.peerconnection)
			room.AddConnectionToRoom(s.peerconnection, s.PlayerIndex)
		}
		s.RoomID = room.ID

		// Register room to overlord if we are connecting to overlord
		if room != nil && s.OverlordClient != nil {
			s.OverlordClient.Send(cws.WSPacket{
				ID:   "registerRoom",
				Data: s.RoomID,
			}, nil)
		}
		req.ID = "start"
		req.RoomID = s.RoomID
		req.SessionID = s.ID

		return req
	})

	browserClient.Receive("candidate", func(resp cws.WSPacket) (req cws.WSPacket) {
		// Unuse code
		hi := pionRTC.ICECandidateInit{}
		err := json.Unmarshal([]byte(resp.Data), &hi)
		if err != nil {
			log.Println("[!] Cannot parse candidate: ", err)
		} else {
			// webRTC.AddCandidate(hi)
		}
		req.ID = "candidate"

		return req
	})

}

// NewOverlordClient returns a client connecting to browser. This connection exchanges information between clients and server
func NewBrowserClient(c *websocket.Conn) *BrowserClient {
	return &BrowserClient{
		Client: cws.NewClient(c),
	}
}
