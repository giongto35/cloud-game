package coordinator

import (
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
	Find(com.Uid) *User
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
	return w.ProcessPackets(func(p api.In[com.Uid]) error {
		payload := p.GetPayload()
		switch p.GetType() {
		case api.RegisterRoom:
			rq := api.Unwrap[api.RegisterRoomRequest](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			w.log.Info().Msgf("set room [%v] = %v", w.Id(), *rq)
			w.HandleRegisterRoom(*rq)
		case api.CloseRoom:
			rq := api.Unwrap[api.CloseRoomRequest](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			w.HandleCloseRoom(*rq)
		case api.IceCandidate:
			rq := api.Unwrap[api.WebrtcIceCandidateRequest[com.Uid]](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			err := w.HandleIceCandidate(*rq, users)
			if err != nil {
				w.log.Error().Err(err).Send()
				return api.ErrMalformed
			}
		case api.LibNewGameList:
			inf := api.Unwrap[api.LibGameListInfo](payload)
			if inf == nil {
				return api.ErrMalformed
			}
			if err := w.HandleLibGameList(*inf); err != nil {
				w.log.Error().Err(err).Send()
				return api.ErrMalformed
			}
		case api.PrevSessions:
			sess := api.Unwrap[api.PrevSessionInfo](payload)
			if sess == nil {
				return api.ErrMalformed
			}
			if err := w.HandlePrevSessionList(*sess); err != nil {
				w.log.Error().Err(err).Send()
				return api.ErrMalformed
			}
		default:
			w.log.Warn().Msgf("Unknown packet: %+v", p)
		}
		return nil
	})
}

func (w *Worker) SetLib(list []api.GameInfo) {
	w.Lib = list
}

func (w *Worker) AppNames() []api.GameInfo {
	return w.Lib
}

func (w *Worker) AddSession(id string) {
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

// Reserve increments user counter of the worker.
func (s *slotted) Reserve() { atomic.AddInt32((*int32)(s), 1) }

// UnReserve decrements user counter of the worker.
func (s *slotted) UnReserve() {
	if atomic.AddInt32((*int32)(s), -1) < 0 {
		atomic.StoreInt32((*int32)(s), 0)
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
