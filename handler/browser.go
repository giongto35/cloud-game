package handler

import (
	"encoding/json"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/handler/gamelist"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"
	uuid "github.com/satori/go.uuid"
)

type BrowserClient struct {
	*cws.Client
	session     *Session
	oclient     *OverlordClient
	gameName    string
	roomID      string
	playerIndex int
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
			SessionID: s.SessionID,
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
		gameName = resp.Data
		roomID = resp.RoomID
		playerIndex = resp.PlayerIndex
		isNewRoom := false

		log.Println("Starting game")
		// If we are connecting to overlord, request serverID from roomID
		if browserClient.oclient != nil {
			session := browserClient.session
			roomServerID := getServerIDOfRoom(session.OverlordClient, roomID)
			log.Println("Server of RoomID ", roomID, " is ", roomServerID)
			if roomServerID != "" && wssession.ServerID != roomServerID {
				// TODO: Re -register
				go bridgeConnection(wssession, roomServerID, gameName, roomID, playerIndex)
				return
			}
		}

		roomID, isNewRoom = startSession(wssession.peerconnection, gameName, roomID, playerIndex)
		// Register room to overlord if we are connecting to overlord
		if isNewRoom && browserClient.session.OverlordClient != nil {
			browserClient.session.OverlordClient.Send(cws.WSPacket{
				ID:   "registerRoom",
				Data: roomID,
			}, nil)
		}
		req.ID = "start"
		req.RoomID = roomID
		req.SessionID = sessionID

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
func NewBrowserClient(c *websocket.Conn, overlordClient *OverlordClient) *BrowserClient {
	roomID := ""
	gameName := ""
	playerIndex := 0
	// Create connection to overlord
	browserClient := &BrowserClient{
		Client:      cws.NewClient(c),
		gameName:    "",
		roomID:      "",
		playerIndex: 0,
	}

	//sessionID := strconv.Itoa(rand.Int())
	sessionID := uuid.Must(uuid.NewV4()).String()

	wssession := &Session{
		BrowserClient:  browserClient,
		OverlordClient: overlordClient,
		peerconnection: webrtc.NewWebRTC(),
		// The server session is maintaining
	}

	browserClient.Send(cws.WSPacket{
		ID:   "gamelist",
		Data: gamelist.GetEncodedGameList(),
	}, nil)
	return browserClient
}
