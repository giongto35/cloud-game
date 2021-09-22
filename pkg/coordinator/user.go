package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

type User struct {
	client.DefaultClient

	RoomID string
	Worker *Worker
}

func NewUser(conn *ipc.Client) User {
	return User{DefaultClient: client.New(conn, "user")}
}

func (u *User) AssignWorker(w *Worker) {
	u.Worker = w
	w.ChangeUserQuantityBy(1)
}

func (u *User) RetainWorker() { u.Worker.ChangeUserQuantityBy(-1) }

func (u *User) AssignRoom(id string) { u.RoomID = id }

func (u *User) HandleRequests(launcher launcher.Launcher) {
	u.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.WebrtcInit:
			u.Printf("Received init_webrtc request -> relay to worker: %s", u.Worker.Id())
			// initWebrtc now only sends signal to worker, asks it to createOffer
			// relay request to target worker
			// worker creates a PeerConnection, and createOffer
			// send SDP back to browser
			u.HandleWebrtcInit()
			u.Printf("Received SDP from worker -> sending back to browser")
		case api.WebrtcAnswer:
			u.Printf("Received browser answered SDP -> relay to worker")
			u.HandleWebrtcAnswer(p.Payload)
		case api.WebrtcIceCandidate:
			u.Printf("Received IceCandidate from browser -> relay to worker")
			u.HandleWebrtcIceCandidate(p.Payload)
		case api.StartGame:
			u.Printf("Received start request from a browser -> relay to worker")
			u.HandleStartGame(p.Payload, launcher)
		case api.QuitGame:
			u.Printf("Received quit request from a browser -> relay to worker")
			u.HandleQuitGame(p.Payload)
		case api.SaveGame:
			u.Printf("Received save request from a browser -> relay to worker")
			u.HandleSaveGame()
		case api.LoadGame:
			u.Printf("Received load request from a browser -> relay to worker")
			u.HandleLoadGame()
		case api.ChangePlayer:
			u.Printf("Received update player index request from a browser -> relay to worker")
			u.HandleChangePlayer(p.Payload)
		case api.ToggleMultitap:
			u.Printf("Received multitap request from a browser -> relay to worker")
			u.HandleToggleMultitap()
		}
	})
}
