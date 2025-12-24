package coordinator

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

type Worker struct {
	AppLibrary
	Connection
	RegionalClient
	Session
	slotted

	Addr       string
	PingServer string
	Port       string
	RoomId     string // room reference
	Tag        string
	Zone       string

	Lib      []api.GameInfo
	Sessions map[string]struct{}

	log *logger.Logger
}

type RegionalClient interface {
	In(region string) bool
}

type HasUserRegistry interface {
	Find(id string) *User
}

type AppLibrary interface {
	SetLib([]api.GameInfo)
	AppNames() []api.GameInfo
}

type Session interface {
	AddSession(id string)
	// HadSession is true when an old session is found
	HadSession(id string) bool
	SetSessions(map[string]struct{})
}

type AppMeta struct {
	Alias  string
	Base   string
	Name   string
	Path   string
	System string
	Type   string
}

func NewWorker(sock *com.Connection, handshake api.ConnectionRequest[com.Uid], log *logger.Logger) *Worker {
	conn := com.NewConnection[api.PT, api.In[com.Uid], api.Out, *api.Out](sock, handshake.Id, log)
	return &Worker{
		Connection: conn,
		Addr:       handshake.Addr,
		PingServer: handshake.PingURL,
		Port:       handshake.Port,
		Tag:        handshake.Tag,
		Zone:       handshake.Zone,
		log: log.Extend(log.With().
			Str(logger.ClientField, logger.MarkNone).
			Str(logger.DirectionField, logger.MarkNone).
			Str("cid", conn.Id().Short())),
	}
}

func (w *Worker) HandleRequests(users HasUserRegistry) chan struct{} {
	return w.ProcessPackets(func(p api.In[com.Uid]) (err error) {
		switch p.T {
		case api.RegisterRoom:
			err = api.Do(p, func(d api.RegisterRoomRequest) {
				w.log.Info().Msgf("set room [%v] = %v", w.Id(), d)
				w.HandleRegisterRoom(d)
			})
		case api.CloseRoom:
			err = api.Do(p, w.HandleCloseRoom)
		case api.IceCandidate:
			err = api.DoE(p, func(d api.WebrtcIceCandidateRequest) error {
				return w.HandleIceCandidate(d, users)
			})
		case api.LibNewGameList:
			err = api.DoE(p, w.HandleLibGameList)
		case api.PrevSessions:
			err = api.DoE(p, w.HandlePrevSessionList)
		default:
			w.log.Warn().Msgf("Unknown packet: %+v", p)
		}
		if err != nil && !errors.Is(err, api.ErrMalformed) {
			w.log.Error().Err(err).Send()
			err = api.ErrMalformed
		}
		return
	})
}

func (w *Worker) SetLib(list []api.GameInfo) { w.Lib = list }

func (w *Worker) AppNames() []api.GameInfo {
	return w.Lib
}

func (w *Worker) AddSession(id string) {
	// sessions can be uninitialized until the coordinator pushes them to the worker
	if w.Sessions == nil {
		return
	}

	w.Sessions[id] = struct{}{}
}

func (w *Worker) HadSession(id string) bool {
	_, ok := w.Sessions[id]
	return ok
}

func (w *Worker) SetSessions(sessions map[string]struct{}) {
	w.Sessions = sessions
}

// In say whether some worker from this region (zone).
// Empty region always returns true.
func (w *Worker) In(region string) bool { return region == "" || region == w.Zone }

// slotted used for tracking user slots and the availability.
type slotted int32

// HasSlot checks if the current worker has a free slot to start a new game.
// Workers support only one game at a time, so it returns true in case if
// there are no players in the room (worker).
func (s *slotted) HasSlot() bool { return atomic.LoadInt32((*int32)(s)) == 0 }

// TryReserve reserves the slot only when it's free.
func (s *slotted) TryReserve() bool {
	for {
		current := atomic.LoadInt32((*int32)(s))
		if current != 0 {
			return false
		}
		if atomic.CompareAndSwapInt32((*int32)(s), 0, 1) {
			return true
		}
	}
}

// UnReserve decrements user counter of the worker.
func (s *slotted) UnReserve() {
	for {
		current := atomic.LoadInt32((*int32)(s))
		if current <= 0 {
			// reset to zero
			if current < 0 {
				if atomic.CompareAndSwapInt32((*int32)(s), current, 0) {
					return
				}
				continue
			}

			return
		}

		// Regular decrement for positive values
		newVal := current - 1
		if atomic.CompareAndSwapInt32((*int32)(s), current, newVal) {
			return
		}
	}
}

func (s *slotted) FreeSlots() { atomic.StoreInt32((*int32)(s), 0) }

func (w *Worker) Disconnect() {
	w.Connection.Disconnect()
	w.RoomId = ""
	w.FreeSlots()
}

func (w *Worker) PrintInfo() string {
	return fmt.Sprintf("id: %v, addr: %v, port: %v, zone: %v, ping addr: %v, tag: %v",
		w.Id(), w.Addr, w.Port, w.Zone, w.PingServer, w.Tag)
}
