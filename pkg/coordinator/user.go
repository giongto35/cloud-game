package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type User struct {
	Connection
	w   *Worker // linked worker
	log *logger.Logger
}

type HasServerInfo interface {
	GetServerList() []api.Server
}

func NewUser(conn *com.Connection, log *logger.Logger) *User {
	socket := com.NewConnection[api.PT, api.In[com.Uid], api.Out](conn, com.NewUid(), log)
	return &User{
		Connection: socket,
		log: log.Extend(log.With().
			Str(logger.DirectionField, logger.MarkNone).
			Str("cid", socket.Id().Short())),
	}
}

// Bind reserves a worker for sole user use.
func (u *User) Bind(w *Worker) {
	u.w = w
	u.w.Reserve()
}

func (u *User) Disconnect() {
	u.Connection.Disconnect()
	if u.w != nil {
		u.w.UnReserve()
		u.w.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(info HasServerInfo, launcher games.Launcher, conf coordinator.Config) chan struct{} {
	u.OnPacket(func(x api.In[com.Uid]) error {
		// !to use proper channels
		payload := x.GetPayload()
		switch x.GetType() {
		case api.WebrtcInit:
			if u.w != nil {
				u.HandleWebrtcInit()
			}
		case api.WebrtcAnswer:
			rq := api.Unwrap[api.WebrtcAnswerUserRequest](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcAnswer(*rq)
		case api.WebrtcIce:
			rq := api.Unwrap[api.WebrtcUserIceCandidate](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcIceCandidate(*rq)
		case api.StartGame:
			rq := api.Unwrap[api.GameStartUserRequest](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleStartGame(*rq, launcher, conf)
		case api.QuitGame:
			rq := api.Unwrap[api.GameQuitRequest[com.Uid]](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleQuitGame(*rq)
		case api.SaveGame:
			return u.HandleSaveGame()
		case api.LoadGame:
			return u.HandleLoadGame()
		case api.ChangePlayer:
			rq := api.Unwrap[api.ChangePlayerUserRequest](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleChangePlayer(*rq)
		case api.ToggleMultitap:
			u.HandleToggleMultitap()
		case api.RecordGame:
			if !conf.Recording.Enabled {
				return api.ErrForbidden
			}
			rq := api.Unwrap[api.RecordGameRequest[com.Uid]](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleRecordGame(*rq)
		case api.GetWorkerList:
			u.handleGetWorkerList(conf.Coordinator.Debug, info)
		default:
			u.log.Warn().Msgf("Unknown packet: %+v", x)
		}
		return nil
	})
	return u.Listen()
}
