package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"
)

type BrowserClient struct {
	*cws.Client
	//session     *Session
	//oclient     *OverlordClient
	//gameName    string
	//roomID      string
	//playerIndex int
}

func (s *Session) RegisterBrowserClient() {
	browserClient := s.BrowserClient

	browserClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	browserClient.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received user SDP")
		localSession, err := s.peerconnection.StartClient(resp.Data, config.Width, config.Height)
		if err != nil {
			log.Fatalln(err)
		}

		return cws.WSPacket{
			ID:        "sdp",
			Data:      localSession,
			SessionID: s.ID,
		}
	})

	// TODO: Add save and load
	//browserClient.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Saving game state")
	//req.ID = "save"
	//req.Data = "ok"
	//if roomID != "" {
	////err := rooms[roomID].director.SaveGame()
	//err := browserClient.room.director.SaveGame()
	//if err != nil {
	//log.Println("[!] Cannot save game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	//browserClient.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
	//log.Println("Loading game state")
	//req.ID = "load"
	//req.Data = "ok"
	//if roomID != "" {
	//err := rooms[roomID].director.LoadGame()
	//if err != nil {
	//log.Println("[!] Cannot load game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	browserClient.Receive("start", func(resp cws.WSPacket) (req cws.WSPacket) {
		s.GameName = resp.Data
		s.RoomID = resp.RoomID
		s.PlayerIndex = resp.PlayerIndex

		log.Println("Starting game")
		// If we are connecting to overlord, request serverID from roomID
		if s.OverlordClient != nil {
			roomServerID := getServerIDOfRoom(s.OverlordClient, s.RoomID)
			log.Println("Server of RoomID ", s.RoomID, " is ", s.RoomID)
			if roomServerID != "" && s.ServerID != roomServerID {
				// TODO: Re -register
				go s.bridgeConnection(roomServerID, s.GameName, s.RoomID, s.PlayerIndex)
				return
			}
		}

		room := s.handler.createNewRoom(s.GameName, s.RoomID, s.PlayerIndex)
		room.addConnectionToRoom(s.peerconnection, s.PlayerIndex)
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
		fmt.Println("Response from start", req)

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
	//roomID := ""
	//gameName := ""
	//playerIndex := 0
	// Create connection to overlord
	browserClient := &BrowserClient{
		Client: cws.NewClient(c),
		//gameName:    "",
		//roomID:      "",
		//playerIndex: 0,
	}

	//sessionID := strconv.Itoa(rand.Int())
	//sessionID := uuid.Must(uuid.NewV4()).String()

	//wssession := &Session{
	//BrowserClient:  browserClient,
	//OverlordClient: overlordClient,
	//peerconnection: webrtc.NewWebRTC(),
	//// The server session is maintaining
	//}

	return browserClient
}
