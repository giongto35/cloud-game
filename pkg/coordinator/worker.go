package coordinator

import (
	"log"
	"sync/atomic"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
)

type Worker struct {
	client.SocketClient
	client.RegionalClient

	Address    string
	PingServer string
	users      int32
	Zone       string
}

func NewWorkerClient(conn *ipc.Client) Worker { return Worker{SocketClient: client.New(conn, "w")} }

func (w *Worker) HandleRequests(rooms *client.NetMap, crowd *client.NetMap) {
	w.SocketClient.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.RegisterRoom:
			w.Logf("Received room register call %s", p.Payload)
			w.HandleRegisterRoom(p.Payload, rooms)
			log.Printf("Rooms: %+v", rooms.List())
		case api.CloseRoom:
			w.Logf("Received room close call %s", p.Payload)
			w.HandleCloseRoom(p.Payload, rooms)
			log.Printf("Current room list is: %+v", rooms.List())
		case api.IceCandidate:
			w.Logf("relay IceCandidate to useragent")
			w.HandleIceCandidate(p.Payload, crowd)
		}
	})
}

// In say whether some worker from this region (zone).
// Empty region always returns true.
func (w *Worker) In(region string) bool { return region == "" || region == w.Zone }

// ChangeUserQuantityBy increases or decreases the total number of
// users connected to the current worker.
// We count users to determine when the worker becomes new game ready.
func (w *Worker) ChangeUserQuantityBy(n int) {
	atomic.AddInt32(&w.users, int32(n))
	if atomic.LoadInt32(&w.users) < 0 {
		atomic.StoreInt32(&w.users, 0)
	}
}

// HasGameSlot checks if the current worker has a free slot to start a new game.
// Workers support only one game at a time.
func (w *Worker) HasGameSlot() bool { return atomic.LoadInt32(&w.users) == 0 }
