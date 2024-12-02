package coordinator

import (
	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

type User struct {
	Connection
	w   *Worker // linked worker
	log *logger.Logger
}

type HasServerInfo interface {
	GetServerList() []api.Server
}

func NewUser(sock *com.Connection, log *logger.Logger) *User {
	conn := com.NewConnection[api.PT, api.In[com.Uid], api.Out, *api.Out](sock, com.NewUid(), log)
	return &User{
		Connection: conn,
		log: log.Extend(log.With().
			Str(logger.ClientField, logger.MarkNone).
			Str(logger.DirectionField, logger.MarkNone).
			Str("cid", conn.Id().Short())),
	}
}

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

func (u *User) HandleRequests(info HasServerInfo, conf config.CoordinatorConfig) chan struct{} {
	return u.ProcessPackets(func(x api.In[com.Uid]) error {
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
			u.HandleStartGame(*rq, conf)
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
		case api.ResetGame:
			rq := api.Unwrap[api.ResetGameRequest[com.Uid]](payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleResetGame(*rq)
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
}
