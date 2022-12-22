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

// NewUserConnection supposed to be a bidirectional one.
func NewUserConnection(conn *com.SocketClient) *User { return &User{SocketClient: *conn} }

func (u *User) SetWorker(w *Worker) { u.w = w; u.w.Reserve() }

func (u *User) Disconnect() {
	u.SocketClient.Close()
	if u.w != nil {
		u.w.UnReserve()
		u.w.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(info api.HasServerInfo, launcher games.Launcher, conf coordinator.Config) {
	u.ProcessMessages()
	u.OnPacket(func(x com.In) error {
		// !to use proper channels
		switch x.T {
		case api.WebrtcInit:
			if u.w != nil {
				u.HandleWebrtcInit()
			}
		case api.WebrtcAnswer:
			rq := api.Unwrap[api.WebrtcAnswerUserRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcAnswer(*rq)
		case api.WebrtcIce:
			rq := api.Unwrap[api.WebrtcUserIceCandidate](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcIceCandidate(*rq)
		case api.StartGame:
			rq := api.Unwrap[api.GameStartUserRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleStartGame(*rq, launcher, conf)
		case api.QuitGame:
			rq := api.Unwrap[api.GameQuitRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleQuitGame(*rq)
		case api.SaveGame:
			return u.HandleSaveGame()
		case api.LoadGame:
			return u.HandleLoadGame()
		case api.ChangePlayer:
			rq := api.Unwrap[api.ChangePlayerUserRequest](x.Payload)
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
			rq := api.Unwrap[api.RecordGameRequest](x.Payload)
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
}
