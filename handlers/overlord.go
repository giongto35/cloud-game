package handler

import (
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
)

func createOverlordConnection() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(*config.OverlordHost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewOverlordClient(oc *websocket.Conn) *Client {
	oclient := NewClient(oc)

	// Received from overlord the serverID
	oclient.receive(
		"serverID",
		func(response WSPacket) (request WSPacket) {
			// Stick session with serverID got from overlord
			log.Println("Received serverID ", response.Data)
			serverID = response.Data

			return EmptyPacket
		},
	)

	// Received from overlord the sdp. This is happens when bridging
	// TODO: refactor
	oclient.receive(
		"initwebrtc",
		func(resp WSPacket) (req WSPacket) {
			log.Println("Received a sdp request from overlord")
			log.Println("Start peerconnection from the sdp")
			peerconnection := webrtc.NewWebRTC()
			// init new peerconnection from sessionID
			localSession, err := peerconnection.StartClient(resp.Data, width, height)
			peerconnections[resp.SessionID] = peerconnection

			if err != nil {
				log.Fatalln(err)
			}

			return WSPacket{
				ID:   "sdp",
				Data: localSession,
			}
		},
	)

	// Received start from overlord. This is happens when bridging
	// TODO: refactor
	oclient.receive(
		"start",
		func(resp WSPacket) (req WSPacket) {
			log.Println("Received a start request from overlord")
			log.Println("Add the connection to current room on the host")

			peerconnection := peerconnections[resp.SessionID]
			log.Println("start session")
			roomID, isNewRoom := startSession(peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
			log.Println("Done, sending back")
			// Bridge always access to old room
			// TODO: log warn
			if isNewRoom == true {
				log.Fatal("Bridge should not spawn new room")
			}

			req.ID = "start"
			req.RoomID = roomID
			return req
		},
	)
	// heartbeat to keep pinging overlord. We not ping from server to browser, so we don't call heartbeat in browserClient
	go oclient.heartbeat()
	go oclient.listen()

	return oclient
}
