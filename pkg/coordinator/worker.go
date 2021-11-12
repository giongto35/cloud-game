package coordinator

import (
	"sync/atomic"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type Worker struct {
	client.SocketClient
	client.RegionalClient

	Address    string
	PingServer string
	users      int32
	log        *logger.Logger
	Zone       string
}

func NewWorkerClient(conn *ipc.Client, log *logger.Logger) Worker {
	c := client.New(conn, "w", log)
	defer c.GetLogger().Info().Msg("Connect")
	return Worker{SocketClient: c, log: c.GetLogger()}
}

func (w *Worker) HandleRequests(rooms *client.NetMap, crowd *client.NetMap) {
	w.SocketClient.OnPacket(func(p ipc.InPacket) {
		go func() {
			switch p.T {
			case api.RegisterRoom:
				w.log.Debug().Msgf("Received room register call %s", p.Payload)
				w.HandleRegisterRoom(p.Payload, rooms)
				w.log.Debug().Msgf("Rooms: %+v", rooms.List())
			case api.CloseRoom:
				w.log.Debug().Msgf("Received room close call %s", p.Payload)
				w.HandleCloseRoom(p.Payload, rooms)
				w.log.Debug().Msgf("Current room list is: %+v", rooms.List())
			case api.IceCandidate:
				w.log.Debug().Msgf("Pass ICE candidate to a user")
				w.HandleIceCandidate(p.Payload, crowd)
			default:
				w.log.Warn().Msgf("Unknown packet: %+v", p)
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
	atomic.AddInt32(&w.users, int32(n))
	if atomic.LoadInt32(&w.users) < 0 {
		atomic.StoreInt32(&w.users, 0)
	}
}

// HasGameSlot checks if the current worker has a free slot to start a new game.
// Workers support only one game at a time.
func (w *Worker) HasGameSlot() bool { return atomic.LoadInt32(&w.users) == 0 }

func (w *Worker) Close() {
	w.SocketClient.Close()
	w.log.Info().Msg("Disconnect")
}
