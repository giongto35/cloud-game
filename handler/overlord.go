package handler

import (
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
)

// OverlordClient maintans connection to overlord
// We expect only one OverlordClient for each server
type OverlordClient struct {
	*cws.Client
	peerconnections map[string]*webrtc.WebRTC
}

// NewOverlordClient returns a client connecting to overlord for coordiation between different server
func NewOverlordClient(oc *websocket.Conn) *OverlordClient {
	if oc == nil {
		return nil
	}

	oclient := &OverlordClient{
		Client: cws.NewClient(oc),
	}
	return oclient
}

// RegisterOverlordClient routes overlord Client
func (s *Session) RegisterOverlordClient() {
	oclient := s.OverlordClient

	// Received from overlord the serverID
	oclient.Receive(
		"serverID",
		func(response cws.WSPacket) (request cws.WSPacket) {
			// Stick session with serverID got from overlord
			log.Println("Received serverID ", response.Data)
			s.ServerID = response.Data

			return cws.EmptyPacket
		},
	)

	// Received from overlord the sdp. This is happens when bridging
	// TODO: refactor
	oclient.Receive(
		"initwebrtc",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a sdp request from overlord")
			log.Println("Start peerconnection from the sdp")
			peerconnection := webrtc.NewWebRTC()
			// init new peerconnection from sessionID
			localSession, err := peerconnection.StartClient(resp.Data, config.Width, config.Height)
			oclient.peerconnections[resp.SessionID] = peerconnection

			if err != nil {
				log.Fatalln(err)
			}

			return cws.WSPacket{
				ID:   "sdp",
				Data: localSession,
			}
		},
	)

	// Received start from overlord. This happens when bridging
	// TODO: refactor
	oclient.Receive(
		"start",
		func(resp cws.WSPacket) (req cws.WSPacket) {
			log.Println("Received a start request from overlord")
			log.Println("Add the connection to current room on the host")

			peerconnection := oclient.peerconnections[resp.SessionID]
			log.Println("start session")

			//room := s.handler.createNewRoom(s.GameName, s.RoomID, s.PlayerIndex)
			// Request room from Server if roomID is existed on the server
			room := s.handler.getRoom(s.RoomID)
			if room == nil {
				log.Println("Room not found", s.RoomID)
				return cws.EmptyPacket
			}
			room.addConnectionToRoom(peerconnection, s.PlayerIndex)
			//roomID, isNewRoom := startSession(peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
			log.Println("Done, sending back")
			// Bridge always access to old room
			// TODO: log warn
			if room != nil {
				log.Fatal("Bridge should not spawn new room")
			}

			req.ID = "start"
			req.RoomID = room.ID
			return req
		},
	)
	// heartbeat to keep pinging overlord. We not ping from server to browser, so we don't call heartbeat in browserClient
}

func getServerIDOfRoom(oc *OverlordClient, roomID string) string {
	log.Println("Request overlord roomID")
	packet := oc.SyncSend(
		cws.WSPacket{
			ID:   "getRoom",
			Data: roomID,
		},
	)
	log.Println("Received roomID from overlord ", packet.Data)

	return packet.Data
}

func (s *Session) bridgeConnection(serverID string, gameName string, roomID string, playerIndex int) {
	log.Println("Bridging connection to other Host ", serverID)
	client := s.BrowserClient
	// Ask client to init

	log.Println("Requesting offer to browser", serverID)
	resp := client.SyncSend(cws.WSPacket{
		ID:   "requestOffer",
		Data: "",
	})

	log.Println("Sending offer to overlord to relay message to target host", resp.TargetHostID)
	// Ask overlord to relay SDP packet to serverID
	resp.TargetHostID = serverID
	remoteTargetSDP := s.OverlordClient.SyncSend(resp)
	log.Println("Got back remote host SDP, sending to browser")
	// Send back remote SDP of remote server to browser
	s.BrowserClient.Send(cws.WSPacket{
		ID:   "sdp",
		Data: remoteTargetSDP.Data,
	}, nil)
	log.Println("Init session done, start game on target host")

	s.OverlordClient.SyncSend(cws.WSPacket{
		ID:           "start",
		Data:         gameName,
		TargetHostID: serverID,
		RoomID:       roomID,
		PlayerIndex:  playerIndex,
	})
	log.Println("Game is started on remote host")
}
