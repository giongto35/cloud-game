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

func (u *User) Bind(w *Worker) bool {
	u.w = w
	// Binding only links the worker; slot reservation is handled lazily on
	// game start to avoid blocking deep-link joins or parallel connections
	// that haven't started a game yet.
	return true
}

func (u *User) Disconnect() {
	u.Connection.Disconnect()
	if u.w != nil {
		u.w.TerminateSession(u.Id().String())
	}
}

func (u *User) HandleRequests(info HasServerInfo, conf config.CoordinatorConfig) chan struct{} {
	return u.ProcessPackets(func(x api.In[com.Uid]) (err error) {
		switch x.T {
		case api.WebrtcInit:
			if u.w != nil {
				u.HandleWebrtcInit()
			}
		case api.WebrtcAnswer:
			err = api.Do(x, u.HandleWebrtcAnswer)
		case api.WebrtcIce:
			err = api.Do(x, u.HandleWebrtcIceCandidate)
		case api.StartGame:
			err = api.Do(x, func(d api.GameStartUserRequest) { u.HandleStartGame(d, conf) })
		case api.QuitGame:
			err = api.Do(x, u.HandleQuitGame)
		case api.SaveGame:
			err = u.HandleSaveGame()
		case api.LoadGame:
			err = u.HandleLoadGame()
		case api.ChangePlayer:
			err = api.Do(x, u.HandleChangePlayer)
		case api.ResetGame:
			err = api.Do(x, u.HandleResetGame)
		case api.RecordGame:
			if !conf.Recording.Enabled {
				return api.ErrForbidden
			}
			err = api.Do(x, u.HandleRecordGame)
		case api.GetWorkerList:
			u.handleGetWorkerList(conf.Coordinator.Debug, info)
		default:
			u.log.Warn().Msgf("Unknown packet: %+v", x)
		}
		return
	})
}
