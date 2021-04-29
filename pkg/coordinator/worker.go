package coordinator

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/cache"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type Worker struct {
	id      network.Uid
	Address string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer string
	//StunTurnServer string
	IsFree bool
	Region string
	wire   *ipc.Client
}

func NewWorker(conn *ipc.Client) Worker {
	return Worker{
		id:     network.NewUid(),
		IsFree: true,
		wire:   conn,
	}
}

func (w *Worker) Id() network.Uid { return w.id }

func (w *Worker) Send(t uint8, data interface{}) (json.RawMessage, error) {
	return w.wire.Call(t, data)
}

func (w *Worker) SendAndForget(t uint8, data interface{}) error {
	return w.wire.Send(t, data)
}

func (w *Worker) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("worker: [%s] %s", w.id.Short(), format), args...)
}

func (w *Worker) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("worker: [%s] %s", w.id.Short(), fmt.Sprint(args...)))
}

func (w *Worker) HandleRequests(rooms *cache.Cache, crowd *cache.Cache) {
	w.wire.OnPacket = func(p ipc.InPacket) {
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
			w.Println("relay IceCandidate to useragent")
			w.HandleIceCandidate(p.Payload, crowd)
		}
	}
}

// InRegion say whether some worker from this region.
// Empty region always returns true.
func (w *Worker) InRegion(region string) bool { return region == "" && region == w.Region }

func (w *Worker) MakeAvailable(avail bool) { w.IsFree = avail }

func (w *Worker) Clean() {
	w.wire.Close()
}
