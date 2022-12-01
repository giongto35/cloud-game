package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

type User struct {
	com.SocketClient

	RoomID string
	Worker *Worker
}

type ServerInfo interface {
	getServerList() []api.Server
}

func NewUserClientServer(conn *com.SocketClient, err error) (*User, error) {
	if err != nil {
		return nil, err
	}
	return &User{SocketClient: *conn}, nil
}

func (u *User) SetRoom(id string) { u.RoomID = id }

func (u *User) SetWorker(w *Worker) {
	u.Worker = w
	u.Worker.SetSlots(-1)
}

func (u *User) Disconnect() {
	u.SocketClient.Close()
	if u.Worker == nil {
		return
	}
	u.Worker.SetSlots(+1)
	if u.Worker != nil {
		u.Worker.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(info ServerInfo, launcher launcher.Launcher, conf coordinator.Config) {
	u.OnPacket(func(x com.In) error {
		// !to use proper channels
		switch x.T {
		case api.WebrtcInit:
			if u.Worker == nil {
				return nil
			}
			u.HandleWebrtcInit()
		case api.WebrtcAnswer:
			rq := api.Unwrap[api.WebrtcAnswerUserRequest](x.Payload)
			if rq == nil {
				return api.ErrMalformed
			}
			u.HandleWebrtcAnswer(*rq)
		case api.WebrtcIceCandidate:
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
