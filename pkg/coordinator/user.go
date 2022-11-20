package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type User struct {
	client.SocketClient

	RoomID string
	Worker *Worker
	log    *logger.Logger
}

func NewUserClient(conn *ipc.Client, log *logger.Logger) User {
	c := client.New(conn, "u", log)
	defer c.GetLogger().Info().Msg("Connect")
	return User{SocketClient: c, log: c.GetLogger()}
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
	u.OnPacket(func(p ipc.InPacket) {
		// !to use proper channels
		go func() {
			switch p.T {
			case api.WebrtcInit:
				u.log.Info().Msgf("Received WebRTC init request -> relay to worker: %s", u.Worker.Id())
				u.HandleWebrtcInit()
				u.log.Info().Msg("Received SDP from worker -> sending back to browser")
			case api.WebrtcAnswer:
				u.log.Info().Msg("Received browser answered SDP -> relay to worker")
				rq, err := api.Unwrap[api.WebrtcAnswerUserRequest](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed WebRTC answer request")
					return
				}
				u.HandleWebrtcAnswer(*rq)
			case api.WebrtcIceCandidate:
				u.log.Info().Msg("Received IceCandidate from browser -> relay to worker")
				rq, err := api.Unwrap[api.WebrtcUserIceCandidate](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed Ice candidate request")
					return
				}
				u.HandleWebrtcIceCandidate(*rq)
			case api.StartGame:
				u.log.Info().Msg("Received start request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.GameStartUserRequest](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed game start request")
					return
				}
				u.HandleStartGame(*rq, launcher, conf)
			case api.QuitGame:
				u.log.Info().Msg("Received quit request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.GameQuitRequest](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed game quit request")
					return
				}
				u.HandleQuitGame(*rq)
			case api.SaveGame:
				u.log.Info().Msg("Received save request from a browser -> relay to worker")
				u.HandleSaveGame()
			case api.LoadGame:
				u.log.Info().Msg("Received load request from a browser -> relay to worker")
				u.HandleLoadGame()
			case api.ChangePlayer:
				u.log.Info().Msg("Received update player index request from a browser -> relay to worker")
				rq, err := api.Unwrap[api.ChangePlayerUserRequest](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed player change request")
					return
				}
				u.HandleChangePlayer(*rq)
			case api.ToggleMultitap:
				u.log.Info().Msg("Received multitap request from a browser -> relay to worker")
				u.HandleToggleMultitap()
			case api.RecordGame:
				u.log.Info().Msg("Received record game request from a browser -> relay to worker")
				if !conf.Recording.Enabled {
					u.log.Warn().Msg("Recording should be disabled!")
					return
				}
				rq, err := api.Unwrap[api.RecordGameRequest](p.Payload)
				if err != nil {
					u.log.Error().Err(err).Msg("malformed record game request")
					return
				}
				u.HandleRecordGame(*rq)
			case api.GetWorkerList:
				u.log.Info().Msg("Received get worker list request from a browser -> relay to worker")
				u.handleGetWorkerList(conf.Coordinator.Debug, info)
			default:
				u.log.Warn().Msgf("Unknown packet: %+v", p)
			}
		}()
	})
}

func (u *User) Close() {
	u.SocketClient.Close()
	u.log.Info().Msg("Disconnect")
}
