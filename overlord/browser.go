package overlord

import (
	"log"

	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client
}

// RouteBrowser are all routes server accepts for browser
func (s *Session) RouteBrowser() {
	iceCandidates := [][]byte{}

	browserClient := s.BrowserClient

	browserClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	browserClient.Receive("icecandidate", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received candidates ", resp.Data)

		iceCandidates = append(iceCandidates, []byte(resp.Data))
		return cws.EmptyPacket
	})

	browserClient.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received sdp request from a browser")
		log.Println("Overlord: Relay sdp request from a browser to worker")

		// relay SDP to target worker and get back SDP of the worker
		// TODO: Async
		log.Println("Overlord: serverID: ", s.ServerID, resp.SessionID)
		resp.SessionID = s.ID
		sdp := s.handler.workerClients[s.ServerID].SyncSend(
			resp,
		)

		log.Println("Overlord: Received sdp request from a worker")
		log.Println("Overlord: Sending back sdp to browser")

		return sdp
	})

	browserClient.Receive("quit", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Overlord: Received quit request from a browser")
		log.Println("Overlord: Relay quit request from a browser to worker")

		// TODO: Async
		resp.SessionID = s.ID
		resp = s.handler.workerClients[s.ServerID].SyncSend(
			resp,
		)

		return cws.EmptyPacket
	})

	browserClient.Receive("start", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received start request from a browser")
		log.Println("Overlord: Relay start request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		workerResp := s.handler.workerClients[s.ServerID].SyncSend(
			resp,
		)
		// Response from worker contains initialized roomID. Set roomID to the session
		s.RoomID = workerResp.RoomID
		log.Println("Received room response from browser: ", workerResp.RoomID)

		return workerResp
	})

	browserClient.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Overlord: Received save request from a browser")
		log.Println("Overlord: Relay save request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		resp.RoomID = s.RoomID
		resp = s.handler.workerClients[s.ServerID].SyncSend(
			resp,
		)

		return resp
	})

	browserClient.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Overlord: Received load request from a browser")
		log.Println("Overlord: Relay load request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		resp.RoomID = s.RoomID
		resp = s.handler.workerClients[s.ServerID].SyncSend(
			resp,
		)

		return resp
	})
}

// NewOverlordClient returns a client connecting to browser. This connection exchanges information between clients and server
func NewBrowserClient(c *websocket.Conn) *BrowserClient {
	return &BrowserClient{
		Client: cws.NewClient(c),
	}
}
