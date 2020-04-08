package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/gorilla/websocket"
)

const pingServer = "%s://%s/echo"

type WorkerClient struct {
	*cws.Client
	ServerID string
	Address  string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer     string
	StunTurnServer string
	IsAvailable    bool
	Zone           string
}

// RouteWorker are all routes server received from worker
func (o *Server) RouteWorker(workerClient *WorkerClient) {
	// registerRoom event from a worker, when worker created a new room.
	// RoomID is global so it is managed by coordinator.
	workerClient.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received registerRoom room %s from worker %s", resp.Data, workerClient.ServerID)
		o.roomToWorker[resp.Data] = workerClient.ServerID
		log.Printf("Coordinator: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "registerRoom",
		}
	})

	// closeRoom event from a worker, when worker close a room
	workerClient.Receive("closeRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Coordinator: Received closeRoom room %s from worker %s", resp.Data, workerClient.ServerID)
		delete(o.roomToWorker, resp.Data)
		log.Printf("Coordinator: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "closeRoom",
		}
	})

	// getRoom returns the server ID based on requested roomID.
	workerClient.Receive("getRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Coordinator: Received a getroom request")
		log.Println("Result: ", o.roomToWorker[resp.Data])
		return cws.WSPacket{
			ID:   "getRoom",
			Data: o.roomToWorker[resp.Data],
		}
	})

	workerClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})
}

// NewWorkerClient returns a client connecting to worker. This connection exchanges information between workers and server
func NewWorkerClient(c *websocket.Conn, serverID string, address string, stunturn string, zone, pingServer string) *WorkerClient {
	return &WorkerClient{
		Client:         cws.NewClient(c),
		ServerID:       serverID,
		PingServer:     pingServer,
		Address:        address,
		StunTurnServer: stunturn,
		IsAvailable:    true,
		Zone:           zone,
	}
}
