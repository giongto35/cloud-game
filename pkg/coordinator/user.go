package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

type User struct {
	client.SocketClient

	RoomID string
	Worker *Worker
}

func NewUserClient(conn *ipc.Client) User { return User{SocketClient: client.New(conn, "u")} }

func (u *User) SetRoom(id string) { u.RoomID = id }

func (u *User) SetWorker(w *Worker) {
	u.Worker = w
	w.ChangeUserQuantityBy(1)
}

func (u *User) FreeWorker() {
	u.Worker.ChangeUserQuantityBy(-1)
	if u.Worker != nil {
		u.Worker.TerminateSession(u.Id())
	}
}

func (u *User) HandleRequests(launcher launcher.Launcher) {
	u.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.WebrtcInit:
			u.Logf("Received init_webrtc request -> relay to worker: %s", u.Worker.Id())
			u.HandleWebrtcInit()
			u.Logf("Received SDP from worker -> sending back to browser")
		case api.WebrtcAnswer:
			u.Logf("Received browser answered SDP -> relay to worker")
			u.HandleWebrtcAnswer(p.Payload)
		case api.WebrtcIceCandidate:
			u.Logf("Received IceCandidate from browser -> relay to worker")
			u.HandleWebrtcIceCandidate(p.Payload)
		case api.StartGame:
			u.Logf("Received start request from a browser -> relay to worker")
			u.HandleStartGame(p.Payload, launcher)
		case api.QuitGame:
			u.Logf("Received quit request from a browser -> relay to worker")
			u.HandleQuitGame(p.Payload)
		case api.SaveGame:
			u.Logf("Received save request from a browser -> relay to worker")
			u.HandleSaveGame()
		case api.LoadGame:
			u.Logf("Received load request from a browser -> relay to worker")
			u.HandleLoadGame()
		case api.ChangePlayer:
			u.Logf("Received update player index request from a browser -> relay to worker")
			u.HandleChangePlayer(p.Payload)
		case api.ToggleMultitap:
			u.Logf("Received multitap request from a browser -> relay to worker")
			u.HandleToggleMultitap()
		}
	})
}
