package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
)

type User struct {
	com.SocketClient
	w *Worker // linked worker
}

type HasServerInfo interface {
	GetServerList() []api.Server[com.Uid]
}

// NewUser supposed to be a bidirectional one.
func NewUser(conn *com.SocketClient) *User { return &User{SocketClient: *conn} }

func (u *User) SetWorker(w *Worker) { u.w = w; u.w.Reserve() }

func (u *User) Disconnect() {
	u.SocketClient.Disconnect()
	if u.w != nil {
		u.w.UnReserve()
		u.w.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(info HasServerInfo, launcher games.Launcher, conf coordinator.Config) chan struct{} {
	u.OnPacket(func(x com.In) error {
		// !to use proper channels
		switch x.T {
		case api.WebrtcInit:
			if u.w != nil {
				u.HandleWebrtcInit()
			}
		case api.WebrtcAnswer:
			rq := com.Unwrap[api.WebrtcAnswerUserRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcAnswer(*rq)
		case api.WebrtcIce:
			rq := com.Unwrap[api.WebrtcUserIceCandidate](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcIceCandidate(*rq)
		case api.StartGame:
			rq := com.Unwrap[api.GameStartUserRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleStartGame(*rq, launcher, conf)
		case api.QuitGame:
			rq := com.Unwrap[api.GameQuitRequest[com.Uid]](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleQuitGame(*rq)
		case api.SaveGame:
			return u.HandleSaveGame()
		case api.LoadGame:
			return u.HandleLoadGame()
		case api.ChangePlayer:
			rq := com.Unwrap[api.ChangePlayerUserRequest](x.Payload)
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
			rq := com.Unwrap[api.RecordGameRequest[com.Uid]](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleRecordGame(*rq)
		case api.GetWorkerList:
			u.handleGetWorkerList(conf.Coordinator.Debug, info)
		default:
			u.Log.Warn().Msgf("Unknown packet: %+v", x)
		}
		return nil
	})
	return u.Listen()
}
