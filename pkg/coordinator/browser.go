package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client
	SessionID string
	RoomID    string
	WorkerID  string // TODO: how about pointer to workerClient?
}

// NewCoordinatorClient returns a client connecting to browser.
// This connection exchanges information between browser and coordinator.
func NewBrowserClient(c *websocket.Conn, browserID string) *BrowserClient {
	return &BrowserClient{
		Client:    cws.NewClient(c),
		SessionID: browserID,
	}
}

// Register new log
func (bc *BrowserClient) Printf(format string, args ...interface{}) {
	newFmt := fmt.Sprintf("Browser %s] %s", bc.SessionID, format)
	log.Printf(newFmt, args...)
}

func (bc *BrowserClient) Println(args ...interface{}) {
	msg := fmt.Sprintf("Browser %s] %s", bc.SessionID, fmt.Sprint(args...))
	log.Println(msg)
}

// RouteBrowser
// Register callbacks for connection from browser -> coordinator
func (o *Server) RouteBrowser(client *BrowserClient) {
	/* WebSocket */
	client.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	/* WebRTC */
	client.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		// initwebrtc now only sends signal to worker, asks it to createOffer
		client.Printf("Received initwebrtc request -> relay to worker: %s", client.WorkerID)

		// relay request to target worker
		// worker creates a PeerConnection, and createOffer
		// send SDP back to browser
		resp.SessionID = client.SessionID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		sdp := wc.SyncSend(resp)

		client.Println("Received SDP from worker -> sending back to browser")
		return sdp
	})

	client.Receive("answer", func(resp cws.WSPacket) cws.WSPacket {
		// contains SDP of browser createAnswer
		// forward to worker
		client.Println("Received browser answered SDP -> relay to worker")

		resp.SessionID = client.SessionID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)

		// no need to response
		return cws.EmptyPacket
	})

	client.Receive("candidate", func(resp cws.WSPacket) cws.WSPacket {
		// contains ICE candidate of browser
		// forward to worker
		client.Println("Received IceCandidate from browser -> relay to worker")

		resp.SessionID = client.SessionID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		wc.Send(resp, nil)

		return cws.EmptyPacket
	})

	/* GameLogic */
	client.Receive("quit", func(resp cws.WSPacket) (req cws.WSPacket) {
		client.Println("Received quit request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		// Send but, waiting
		wc.SyncSend(resp)

		return cws.EmptyPacket
	})

	client.Receive("start", func(resp cws.WSPacket) cws.WSPacket {
		client.Println("Received start request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		workerResp := wc.SyncSend(resp)

		// Response from worker contains initialized roomID. Set roomID to the session
		client.RoomID = workerResp.RoomID
		client.Println("Received room response from browser: ", workerResp.RoomID)

		return workerResp
	})

	client.Receive("save", func(resp cws.WSPacket) (req cws.WSPacket) {
		client.Println("Received save request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		resp.RoomID = client.RoomID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	})

	client.Receive("load", func(resp cws.WSPacket) (req cws.WSPacket) {
		client.Println("Received load request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		resp.RoomID = client.RoomID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	})

	client.Receive("playerIdx", func(resp cws.WSPacket) (req cws.WSPacket) {
		client.Println("Received update player index request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		resp.RoomID = client.RoomID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	})

	client.Receive("multitap", func(resp cws.WSPacket) (req cws.WSPacket) {
		client.Println("Received multitap request from a browser -> relay to worker")

		// TODO: Async
		resp.SessionID = client.SessionID
		resp.RoomID = client.RoomID
		wc, ok := o.workerClients[client.WorkerID]
		if !ok {
			return cws.EmptyPacket
		}
		resp = wc.SyncSend(resp)

		return resp
	})
}
