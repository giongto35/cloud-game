package overlord

import (
	"log"

	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client
	ServerID       string
	Address        string
	StunTurnServer string
	IsAvailable    bool
}

// RouteWorker are all routes server received from worker
func (o *Server) RouteWorker(workerClient *WorkerClient) {
	// registerRoom event from a server, when server created a new room.
	// RoomID is global so it is managed by overlord.
	workerClient.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received registerRoom ", resp.Data, workerClient.ServerID)
		o.roomToServer[resp.Data] = workerClient.ServerID
		return cws.WSPacket{
			ID: "registerRoom",
		}
	})

	// getRoom returns the server ID based on requested roomID.
	workerClient.Receive("getRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received a getroom request")
		log.Println("Result: ", o.roomToServer[resp.Data])
		return cws.WSPacket{
			ID:   "getRoom",
			Data: o.roomToServer[resp.Data],
		}
	})

	workerClient.Receive("heartbeat", func(resp cws.WSPacket) cws.WSPacket {
		return resp
	})
}

// NewWorkerClient returns a client connecting to worker. This connection exchanges information between workers and server
func NewWorkerClient(c *websocket.Conn, serverID string, address string, stunturn string) *WorkerClient {
	return &WorkerClient{
		Client:         cws.NewClient(c),
		ServerID:       serverID,
		Address:        address,
		StunTurnServer: stunturn,
		IsAvailable:    true,
	}
}
