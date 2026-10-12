package coordinator

import (
	"sync/atomic"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

type User struct {
	Connection
	w   *Worker // linked worker
	log *logger.Logger

	acc atomic.Pointer[auth.Account]

	// room membership, guarded by Hub.mu
	room     string    // joined room id
	slot     int       // player slot (0-3) or SpectatorSlot
	joined   time.Time // when the user joined the room
	lastChat time.Time
}

type HasServerInfo interface {
	GetServerList() []api.Server
}

func NewUser(sock *com.Connection, acc *auth.Account, log *logger.Logger) *User {
	conn := com.NewConnection[api.PT, api.In[com.Uid], api.Out](sock, com.NewUid(), log)
	u := &User{
		Connection: conn,
		slot:       SpectatorSlot,
		log: log.Extend(log.With().
			Str(logger.ClientField, logger.MarkNone).
			Str(logger.DirectionField, logger.MarkNone).
			Str("cid", conn.Id().Short()).
			Int64("acc", acc.Id)),
	}
	u.acc.Store(acc)
	return u
}

// Account returns the current account of the user.
func (u *User) Account() *auth.Account { return u.acc.Load() }

func (u *User) SetAccount(a *auth.Account) { u.acc.Store(a) }

func (u *User) Info() *api.UserInfo {
	a := u.Account()
	perms := make([]string, 0, len(a.Perms))
	for _, p := range a.Perms.List() {
		perms = append(perms, string(p))
	}
	return &api.UserInfo{Id: a.Id, Name: a.Name, Email: a.Email, Role: string(a.Role), Perms: perms, Primary: a.Primary}
}

func (u *User) Disconnect() {
	u.Connection.Disconnect()
	if u.w != nil {
		u.w.TerminateSession(u.Id().String())
	}
}

func (u *User) HandleRequests(h *Hub, conf config.CoordinatorConfig) chan struct{} {
	return u.ProcessPackets(func(x api.In[com.Uid]) (err error) {
		switch x.T {
		case api.InitWebrtcStream:
			err = api.Do(x, u.HandleInitWebrtcStream)
		case api.WebrtcSignal:
			err = api.Do(x, u.HandleWebrtcSignal)
		case api.StartGame:
			err = api.Do(x, func(d api.GameStartUserRequest) { u.HandleStartGame(d, conf, h) })
		case api.QuitGame:
			err = api.Do(x, func(d api.GameQuitRequest) { u.HandleQuitGame(d, h) })
		case api.SaveGame:
			err = u.HandleSaveGame(h)
		case api.LoadGame:
			err = u.HandleLoadGame(h)
		case api.ChangePlayer:
			err = api.Do(x, func(d api.ChangePlayerUserRequest) { u.HandleChangePlayer(d, h) })
		case api.ResetGame:
			err = api.Do(x, func(d api.ResetGameRequest) { u.HandleResetGame(d, h) })
		case api.RecordGame:
			if !conf.Recording.Enabled {
				return api.ErrForbidden
			}
			err = api.Do(x, u.HandleRecordGame)
		case api.GetWorkerList:
			if !u.Account().Can(auth.PermRoomsModerate) {
				return api.ErrForbidden
			}
			u.handleGetWorkerList(conf.Coordinator.Debug, h)
		case api.QueueJoin:
			err = api.Do(x, func(d api.QueueJoinRequest) { h.QueueJoin(u, d.Game) })
		case api.QueueLeave:
			h.QueueLeave(u)
		case api.ChatSend:
			err = api.Do(x, func(d api.ChatSendRequest) { h.Chat(u, d.Text) })
		case api.RoomKick:
			err = api.Do(x, func(d api.RoomKickRequest) { u.HandleRoomKick(d, h) })
		case api.RoomAssign:
			err = api.Do(x, func(d api.RoomAssignRequest) { u.HandleRoomAssign(d, h) })
		default:
			u.log.Warn().Msgf("Unknown packet: %+v", x)
		}
		return
	})
}

// notice sends a short message to show to the user.
func (u *User) notice(kind, text string) {
	u.Notify(api.Notice, api.NoticeResponse{Kind: kind, Text: text})
}
