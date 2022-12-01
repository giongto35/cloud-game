package coordinator

import (
	"sync/atomic"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
)

type Worker struct {
	com.SocketClient
	com.RegionalClient

	Addr       string
	PingServer string
	Port       string
	Tag        string
	users      int32
	Zone       string
}

func (w *Worker) HandleRequests(rooms *com.NetMap[com.NetClient], users *com.NetMap[*User]) {
	// !to make a proper multithreading abstraction
	w.OnPacket(func(p com.In) error {
		switch p.T {
		case api.RegisterRoom:
			rq := api.Unwrap[api.RegisterRoomRequest](p.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			w.HandleRegisterRoom(*rq, rooms)
		case api.CloseRoom:
			rq := api.Unwrap[api.CloseRoomRequest](p.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			w.HandleCloseRoom(*rq, rooms)
		case api.IceCandidate:
			rq := api.Unwrap[api.WebrtcIceCandidateRequest](p.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			w.HandleIceCandidate(*rq, users)
		default:
			w.Log.Warn().Msgf("Unknown packet: %+v", p)
		}
		return nil
	})
}

// In say whether some worker from this region (zone).
// Empty region always returns true.
func (w *Worker) In(region string) bool { return region == "" || region == w.Zone }

// SetSlots adds or removes user slots of the worker.
// We count users to determine when the worker becomes ready for a game.
func (w *Worker) SetSlots(n int) {
	if atomic.AddInt32(&w.users, -int32(n)) < 0 {
		atomic.StoreInt32(&w.users, 0)
	}
}

// HasGameSlot checks if the current worker has a free slot to start a new game.
// Workers support only one game at a time.
func (w *Worker) HasGameSlot() bool { return atomic.LoadInt32(&w.users) == 0 }

func (w *Worker) Disconnect() { w.SocketClient.Close() }
