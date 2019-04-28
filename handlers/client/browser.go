package client

import (
	"encoding/json"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/handlers/gamelist"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

var rooms = map[string]*Room{}

// NewOverlordClient returns a client connecting to browser. This connection exchanges information between clients and server
func NewBrowserClient(c *websocket.Conn) *cws.Client {
	roomID := ""
	// Create connection to overlord
	client := cws.NewClient(c)
	//sessionID := strconv.Itoa(rand.Int())
	sessionID := uuid.Must(uuid.NewV4()).String()

	wssession := &Session{
		client:         client,
		peerconnection: webrtc.NewWebRTC(),
		// The server session is maintaining
	}

	client.Send(cws.WSPacket{
		ID:   "gamelist",
		Data: gamelist.GetEncodedGameList(),
	}, nil)

	client.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	client.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received user SDP")
		localSession, err := wssession.peerconnection.StartClient(resp.Data, config.Width, config.Height)
		if err != nil {
			log.Fatalln(err)
		}

		return cws.WSPacket{
			ID:        "sdp",
			Data:      localSession,
			SessionID: sessionID,
		}
	})

	client.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Saving game state")
		req.ID = "save"
		req.Data = "ok"
		if roomID != "" {
			err := rooms[roomID].director.SaveGame()
			if err != nil {
				log.Println("[!] Cannot save game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}

		return req
	})

	client.Receive("load", func(resp WSPacket) (req WSPacket) {
		log.Println("Loading game state")
		req.ID = "load"
		req.Data = "ok"
		if roomID != "" {
			err = rooms[roomID].director.LoadGame()
			if err != nil {
				log.Println("[!] Cannot load game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}

		return req
	})

	client.Receive("start", func(resp WSPacket) (req WSPacket) {
		gameName = resp.Data
		roomID = resp.RoomID
		playerIndex = resp.PlayerIndex
		isNewRoom := false

		log.Println("Starting game")
		// If we are connecting to overlord, request serverID from roomID
		if oclient != nil {
			roomServerID := getServerIDOfRoom(oclient, roomID)
			log.Println("Server of RoomID ", roomID, " is ", roomServerID)
			if roomServerID != "" && wssession.ServerID != roomServerID {
				// TODO: Re -register
				go bridgeConnection(wssession, roomServerID, gameName, roomID, playerIndex)
				return
			}
		}

		roomID, isNewRoom = startSession(wssession.peerconnection, gameName, roomID, playerIndex)
		// Register room to overlord if we are connecting to overlord
		if isNewRoom && oclient != nil {
			oclient.send(WSPacket{
				ID:   "registerRoom",
				Data: roomID,
			}, nil)
		}
		req.ID = "start"
		req.RoomID = roomID
		req.SessionID = sessionID

		return req
	})

	client.Receive("candidate", func(resp cws.WSPacket) (req cws.WSPacket) {
		// Unuse code
		hi := pionRTC.ICECandidateInit{}
		err = json.Unmarshal([]byte(resp.Data), &hi)
		if err != nil {
			log.Println("[!] Cannot parse candidate: ", err)
		} else {
			// webRTC.AddCandidate(hi)
		}
		req.ID = "candidate"

		return req
	})

	return client
}
