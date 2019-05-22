package overlord

import (
	"log"

	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client
	ServerID string
}

// RouteWorker are all routes server received from worker
func (o *Server) RouteWorker(workerClient WorkerClient) {
	iceCandidates := [][]byte{}

	// registerRoom event from a server, when server created a new room.
	// RoomID is global so it is managed by overlord.
	workerClient.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received registerRoom ", resp.Data, serverID)
		o.roomToServer[resp.Data] = serverID
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
		log.Println("received heartbeat")
		return resp
	})

	workerClient.Receive("icecandidate", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received candidates ", resp.Data)
		iceCandidates = append(iceCandidates, []byte(resp.Data))
		return cws.EmptyPacket
	})

	workerClient.Receive("sdp", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received sdp request from a worker")
		log.Println("Overlord: Sending back sdp to browser")
		s.BrowserClient.Send(resp, nil)

		return cws.EmptyPacket
	})
}

// NewWorkerClient returns a client connecting to worker. This connection exchanges information between workers and server
func NewWorkerClient(c *websocket.Conn, serverID string) *WorkerClient {
	return &WorkerClient{
		Client:   cws.NewClient(c),
		ServerID: serverID,
	}
}
