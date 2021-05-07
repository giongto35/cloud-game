package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/cache"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
)

type Worker struct {
	client.DefaultClient
	client.RegionalClient

	Address string // ip address of worker
	// public server used for ping check
	PingServer string
	//StunTurnServer string
	IsFree bool
	Region string
}

func NewWorker(conn *ipc.Client) Worker {
	return Worker{
		DefaultClient: client.New(conn, "worker"),
		IsFree:        true,
	}
}

func (w *Worker) HandleRequests(rooms *cache.Cache, crowd *cache.Cache) {
	w.DefaultClient.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.RegisterRoom:
			log.Printf("Received registerRoom room %s from worker %s", p.Payload, w.Id())
			w.HandleRegisterRoom(p.Payload, rooms)
			log.Printf("Current room list is: %+v", rooms.List())
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
func (w *Worker) In(region string) bool { return region == "" || region == w.Region }

func (w *Worker) MakeAvailable(avail bool) { w.IsFree = avail }
