package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

type User struct {
	*client.SocketClient

	RoomID string
	Worker *Worker
}

type ServerInfo interface {
	getServerList() []api.Server
}

func NewUserClientServer(conn *client.SocketClient, err error) (*User, error) {
	if err != nil {
		return nil, err
	}
	return &User{SocketClient: conn}, nil
}

func (u *User) SetRoom(id string) { u.RoomID = id }

func (u *User) SetWorker(w *Worker) {
	u.Worker = w
	u.Worker.ChangeUserQuantityBy(1)
}

func (u *User) FreeWorker() {
	u.Worker.ChangeUserQuantityBy(-1)
	if u.Worker != nil {
		u.Worker.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(info ServerInfo, launcher launcher.Launcher, conf coordinator.Config) {
	u.OnPacket(func(p client.In) {
		// !to use proper channels
		go func() {
			switch p.T {
			case api.WebrtcInit:
				u.Log.Info().Msgf("Received WebRTC init request -> relay to worker: %s", u.Worker.Id())
				u.HandleWebrtcInit()
				u.Log.Info().Msg("Received SDP from worker -> sending back to browser")
			case api.WebrtcAnswer:
				u.Log.Info().Msg("Received browser answered SDP -> relay to worker")
				rq, err := api.Unwrap[api.WebrtcAnswerUserRequest](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed WebRTC answer request")
					return
				}
				u.HandleWebrtcAnswer(*rq)
			case api.WebrtcIceCandidate:
				u.Log.Info().Msg("Received IceCandidate from browser -> relay to worker")
				rq, err := api.Unwrap[api.WebrtcUserIceCandidate](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed Ice candidate request")
					return
				}
				u.HandleWebrtcIceCandidate(*rq)
			case api.StartGame:
				u.Log.Info().Msg("Received start request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.GameStartUserRequest](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed game start request")
					return
				}
				u.HandleStartGame(*rq, launcher, conf)
			case api.QuitGame:
				u.Log.Info().Msg("Received quit request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.GameQuitRequest](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed game quit request")
					return
				}
				u.HandleQuitGame(*rq)
			case api.SaveGame:
				u.Log.Info().Msg("Received save request from a browser -> relay to worker")
				u.HandleSaveGame()
			case api.LoadGame:
				u.Log.Info().Msg("Received load request from a browser -> relay to worker")
				u.HandleLoadGame()
			case api.ChangePlayer:
				u.Log.Info().Msg("Received update player index request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.ChangePlayerUserRequest](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed player change request")
					return
				}
				u.HandleChangePlayer(*rq)
			case api.ToggleMultitap:
				u.Log.Info().Msg("Received multitap request from a browser -> relay to worker")
				u.HandleToggleMultitap()
			case api.RecordGame:
				u.Log.Info().Msg("Received record game request from a browser -> relay to worker")
				if !conf.Recording.Enabled {
					u.Log.Warn().Msg("Recording should be disabled!")
					return
				}
				rq, err := api.Unwrap[api.RecordGameRequest](p.Payload)
				if err != nil {
					u.Log.Error().Err(err).Msg("malformed record game request")
					return
				}
				u.HandleRecordGame(*rq)
			case api.GetWorkerList:
				u.Log.Info().Msg("Received get worker list request from a browser -> relay to worker")
				u.handleGetWorkerList(conf.Coordinator.Debug, info)
			default:
				u.Log.Warn().Msgf("Unknown packet: %+v", p)
			}
		}()
	})
}

func (u *User) Disconnect() {
	u.SocketClient.Close()
	u.FreeWorker()
	u.Log.Info().Msg("Disconnect")
}
