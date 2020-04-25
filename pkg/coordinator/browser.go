package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client
}

// RouteBrowser are all routes server accepts for browser
func (s *Session) RouteBrowser() {
	browserClient := s.BrowserClient

	// websocket
	browserClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	// webrtc
	browserClient.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		// initwebrtc now only sends signal to worker, asks it to createOffer
		log.Println("Coordinator: Received initwebrtc request from a browser")
		log.Println("Coordinator: Relay initwebrtc request from a browser to worker")

		// relay request to target worker
		// worker creates a PeerConnection, and createOffer
		// send SDP back to browser
		// TODO: Async
		log.Println("Coordinator: serverID: ", s.ServerID, resp.SessionID)
		resp.SessionID = s.ID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		sdp := wc.SyncSend(
			resp,
		)

		log.Println("Coordinator: Received sdp request from a worker")
		log.Println("Coordinator: Sending back sdp to browser")

		return sdp
	})

	browserClient.Receive("answer", func(resp cws.WSPacket) cws.WSPacket {
		// SDP of browser createAnswer
		// forward to worker
		log.Println("Coordinator: Received browser answered SDP")
		log.Println("Coordinator: Relay SDP from a browser to worker")

		// TODO: refactor this manual assignment
		resp.SessionID = s.ID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)

		// no need to response
		return cws.EmptyPacket
	})

	browserClient.Receive("candidate", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received icecandidate from a browser", resp.Data)
		log.Println("Coordinator: Relay icecandidate from a browser to worker")

		// TODO: refactor this manual assignment
		resp.SessionID = s.ID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)

		return cws.EmptyPacket
	})

	// game
	browserClient.Receive("quit", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Coordinator: Received quit request from a browser")
		log.Println("Coordinator: Relay quit request from a browser to worker")

		// TODO: Async
		resp.SessionID = s.ID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(
			resp,
		)

		return cws.EmptyPacket
	})

	browserClient.Receive("start", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received start request from a browser")
		log.Println("Coordinator: Relay start request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		workerResp := wc.SyncSend(
			resp,
		)
		// Response from worker contains initialized roomID. Set roomID to the session
		s.RoomID = workerResp.RoomID
		log.Println("Received room response from browser: ", workerResp.RoomID)

		return workerResp
	})

	browserClient.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Coordinator: Received save request from a browser")
		log.Println("Coordinator: Relay save request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		resp.RoomID = s.RoomID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(
			resp,
		)

		return resp
	})

	browserClient.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Coordinator: Received load request from a browser")
		log.Println("Coordinator: Relay load request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		resp.RoomID = s.RoomID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(
			resp,
		)

		return resp
	})

	browserClient.Receive("playerIdx", func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Coordinator: Received update player index request from a browser")
		log.Println("Coordinator: Relay update player index request from a browser to worker")
		// TODO: Async
		resp.SessionID = s.ID
		resp.RoomID = s.RoomID
		wc, ok := s.handler.workerClients[s.ServerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(
			resp,
		)

		return resp
	})
}

// NewCoordinatorClient returns a client connecting to browser. This connection exchanges information between clients and server
func NewBrowserClient(c *websocket.Conn) *BrowserClient {
	return &BrowserClient{
		Client: cws.NewClient(c),
	}
}
