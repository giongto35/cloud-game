package overlord

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/gorilla/websocket"
)

const publicWorkerTemplate = "%s/ping/%s"

type WorkerClient struct {
	*cws.Client
	ServerID       string
	Address        string // ip address of worker
	PublicDomain   string // public domain of worker, used for serving echo route
	StunTurnServer string
	IsAvailable    bool
	Zone           string
}

// RouteWorker are all routes server received from worker
func (o *Server) RouteWorker(workerClient *WorkerClient) {
	// registerRoom event from a worker, when worker created a new room.
	// RoomID is global so it is managed by overlord.
	workerClient.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Overlord: Received registerRoom room %s from worker %s", resp.Data, workerClient.ServerID)
		o.roomToWorker[resp.Data] = workerClient.ServerID
		log.Printf("Overlord: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "registerRoom",
		}
	})

	// closeRoom event from a worker, when worker close a room
	workerClient.Receive("closeRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("Overlord: Received closeRoom room %s from worker %s", resp.Data, workerClient.ServerID)
		delete(o.roomToWorker, resp.Data)
		log.Printf("Overlord: Current room list is: %+v", o.roomToWorker)

		return cws.WSPacket{
			ID: "closeRoom",
		}
	})

	// getRoom returns the server ID based on requested roomID.
	workerClient.Receive("getRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received a getroom request")
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
func NewWorkerClient(c *websocket.Conn, serverID string, address string, domain string, stunturn string, zone string) *WorkerClient {
	return &WorkerClient{
		Client:         cws.NewClient(c),
		ServerID:       serverID,
		Address:        address,
		PublicDomain:   fmt.Sprintf(publicWorkerTemplate, domain, serverID),
		StunTurnServer: stunturn,
		IsAvailable:    true,
		Zone:           zone,
	}
}
