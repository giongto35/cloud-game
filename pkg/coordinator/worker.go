package coordinator

import (
	"log"
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
)

type Worker struct {
	client.DefaultClient
	client.RegionalClient

	Address    string // ip address of worker
	PingServer string
	userCount  int // may be atomic
	Zone       string

	mu sync.Mutex
}

func NewWorker(conn *ipc.Client) Worker { return Worker{DefaultClient: client.New(conn, "w")} }

func (w *Worker) HandleRequests(rooms *client.NetMap, crowd *client.NetMap) {
	w.DefaultClient.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.RegisterRoom:
			log.Printf("Received registerRoom room %s from worker %s", p.Payload, w.Id())
			w.HandleRegisterRoom(p.Payload, rooms)
			log.Printf("Rooms: %+v", rooms.List())
		case api.CloseRoom:
			log.Printf("Received closeRoom room %s from worker %s", p.Payload, w.Id())
			w.HandleCloseRoom(p.Payload, rooms)
			log.Printf("Current room list is: %+v", rooms.List())
		case api.IceCandidate:
			w.Printf("relay IceCandidate to useragent")
			w.HandleIceCandidate(p.Payload, crowd)
		}
	})
}

// In say whether some worker from this region.
// Empty region always returns true.
func (w *Worker) In(region string) bool { return region == "" || region == w.Zone }

// ChangeUserQuantityBy increases or decreases the total amount of
// users connected to the current worker.
// We count users to determine when the worker becomes new game ready.
func (w *Worker) ChangeUserQuantityBy(n int) {
	w.mu.Lock()
	w.userCount += n
	// just to be on a safe side
	if w.userCount < 0 {
		w.userCount = 0
	}
	w.mu.Unlock()
}

// HasGameSlot tells whether the current worker has a
// free slot to start a new game.
// Workers support only one game at a time.
func (w *Worker) HasGameSlot() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.userCount == 0
}
