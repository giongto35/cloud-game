package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

const pingServer = "%s://%s/echo"

type WorkerClient struct {
	*cws.Client
	WorkerID string
	Address  string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer     string
	StunTurnServer string
	IsAvailable    bool
	Zone           string
}

// NewWorkerClient returns a client connecting to worker. This connection exchanges information between workers and server
func NewWorkerClient(c *websocket.Conn, workerID string) *WorkerClient {
	return &WorkerClient{
		Client:      cws.NewClient(c),
		WorkerID:    workerID,
		IsAvailable: true,
	}
}

// Register new log
func (wc *WorkerClient) Printf(format string, args ...interface{}) {
	newFmt := fmt.Sprintf("Worker %s] %s", wc.WorkerID, format)
	log.Printf(newFmt, args...)
}

func (wc *WorkerClient) Println(args ...interface{}) {
	msg := fmt.Sprintf("Worker %s] %s", wc.WorkerID, fmt.Sprint(args...))
	log.Println(msg)
}

// RouteWorker are all routes server received from worker
func (o *Server) RouteWorker(wc *WorkerClient) {
	// registerRoom event from a worker, when worker created a new room.
	// RoomID is global so it is managed by coordinator.
	wc.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received registerRoom room %s from worker %s", resp.Data, wc.WorkerID)
		o.roomToWorker[resp.Data] = wc.WorkerID
		log.Printf("Coordinator: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "registerRoom",
		}
	})

	// closeRoom event from a worker, when worker close a room
	wc.Receive("closeRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received closeRoom room %s from worker %s", resp.Data, wc.WorkerID)
		delete(o.roomToWorker, resp.Data)
		log.Printf("Coordinator: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "closeRoom",
		}
	})

	// getRoom returns the server ID based on requested roomID.
	wc.Receive("getRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received a getroom request")
		log.Println("Result: ", o.roomToWorker[resp.Data])
		return cws.WSPacket{
			ID:   "getRoom",
			Data: o.roomToWorker[resp.Data],
		}
	})

	wc.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})

	/* WebRTC */
	wc.Receive("candidate", func(resp cws.WSPacket) cws.WSPacket {
		wc.Println("Received IceCandidate from worker -> relay to browser")
		bc, ok := o.browserClients[resp.SessionID]
		if ok {
			// Remove SessionID while sending back to browser
			resp.SessionID = ""
			bc.Send(resp, nil)
		} else {
			wc.Println("Error: unknown SessionID:", resp.SessionID)
		}

		return cws.EmptyPacket
	})
}
