package coordinator

import (
	"sync/atomic"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/comm"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/rs/xid"
)

type Worker struct {
	*comm.SocketClient
	comm.RegionalClient

	Addr       string
	PingServer string
	Port       string
	Tag        string
	users      int32
	Zone       string
}

func NewWorkerClientServer(id network.Uid, conn *comm.SocketClient) *Worker {
	if id != "" {
		if _, err := xid.FromString(string(id)); err != nil {
			id = network.NewUid()
		}
	} else {
		id = network.NewUid()
	}
	defer conn.Log.Info().Msg("Connect")
	return &Worker{SocketClient: conn}
}

func (w *Worker) HandleRequests(rooms *comm.NetMap, crowd *comm.NetMap) {
	w.SocketClient.OnPacket(func(p comm.In) {
		go func() {
			switch p.T {
			case api.RegisterRoom:
				w.Log.Debug().Msgf("Received room register call %s", p.Payload)
				rq := api.Unwrap[api.RegisterRoomRequest](p.Payload)
				if rq == nil {
					w.Log.Error().Msg("malformed room register request")
					return
				}
				w.HandleRegisterRoom(*rq, rooms)
				w.Log.Debug().Msgf("Rooms: %+v", rooms.List())
			case api.CloseRoom:
				w.Log.Debug().Msgf("Received room close call %s", p.Payload)
				rq := api.Unwrap[api.CloseRoomRequest](p.Payload)
				if rq == nil {
					w.Log.Error().Msg("malformed room remove request")
					return
				}
				w.HandleCloseRoom(*rq, rooms)
				w.Log.Debug().Msgf("Current room list is: %+v", rooms.List())
			case api.IceCandidate:
				w.Log.Debug().Msgf("Pass ICE candidate to a user")
				rq := api.Unwrap[api.WebrtcIceCandidateRequest](p.Payload)
				if rq == nil {
					w.Log.Error().Msg("malformed Ice candidate request")
					return
				}
				w.HandleIceCandidate(*rq, crowd)
			default:
				w.Log.Warn().Msgf("Unknown packet: %+v", p)
			}
		}()
	})
}

// In say whether some worker from this region (zone).
// Empty region always returns true.
func (w *Worker) In(region string) bool { return region == "" || region == w.Zone }

// ChangeUserQuantityBy increases or decreases the total number of
// users connected to the current worker.
// We count users to determine when the worker becomes new game ready.
func (w *Worker) ChangeUserQuantityBy(n int) {
	if atomic.AddInt32(&w.users, int32(n)) < 0 {
		atomic.StoreInt32(&w.users, 0)
	}
}

// HasGameSlot checks if the current worker has a free slot to start a new game.
// Workers support only one game at a time.
func (w *Worker) HasGameSlot() bool { return atomic.LoadInt32(&w.users) == 0 }

func (w *Worker) Disconnect() {
	w.SocketClient.Close()
	w.Log.Info().Msg("Disconnect")
}
