package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type User struct {
	id     network.Uid
	RoomID string
	wire   *ipc.Client
	Worker *Worker
}

func (u *User) Id() network.Uid {
	return u.id
}

func (u *User) InRegion(region string) bool {
	panic("implement me")
}

func NewUser(conn *ipc.Client) User {
	return User{id: network.NewUid(), wire: conn}
}

func (u *User) AssignWorker(w *Worker) {
	u.Worker = w
	w.MakeAvailable(false)
}

func (u *User) RetainWorker() { u.Worker.MakeAvailable(true) }

func (u *User) AssignRoom(id string) { u.RoomID = id }

func (u *User) Send(t uint8, data interface{}) (interface{}, error) {
	return u.wire.Call(t, data)
}

func (u *User) SendAndForget(t uint8, data interface{}) error {
	return u.wire.Send(t, data)
}

func (u *User) WaitDisconnect() {
	<-u.wire.Conn.Done
}

func (u *User) Clean() {
	u.wire.Close()
}

func (u *User) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("user: [%s] %s", u.id.Short(), format), args...)
}

func (u *User) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("user: [%s] %s", u.id.Short(), fmt.Sprint(args...)))
}

func (u *User) HandleRequests(launcher launcher.Launcher) {
	u.wire.OnPacket = func(p ipc.InPacket) {
		switch p.T {
		case api.WebrtcInit:
			u.Printf("Received init_webrtc request -> relay to worker: %s", u.Worker.Id())
			// initWebrtc now only sends signal to worker, asks it to createOffer
			// relay request to target worker
			// worker creates a PeerConnection, and createOffer
			// send SDP back to browser
			u.HandleWebrtcInit()
			u.Println("Received SDP from worker -> sending back to browser")
		case api.WebrtcAnswer:
			u.Println("Received browser answered SDP -> relay to worker")
			u.HandleWebrtcAnswer(p.Payload)
		case api.WebrtcIceCandidate:
			u.Println("Received IceCandidate from browser -> relay to worker")
			u.HandleWebrtcIceCandidate(p.Payload)
		case api.StartGame:
			u.Println("Received start request from a browser -> relay to worker")
			u.HandleStartGame(p.Payload, launcher)
		case api.QuitGame:
			u.Println("Received quit request from a browser -> relay to worker")
			u.HandleQuitGame(p.Payload)
		case api.SaveGame:
			u.Println("Received save request from a browser -> relay to worker")
			u.HandleSaveGame()
		case api.LoadGame:
			u.Println("Received load request from a browser -> relay to worker")
			u.HandleLoadGame()
		case api.ChangePlayer:
			u.Println("Received update player index request from a browser -> relay to worker")
			u.HandleChangePlayer(p.Payload)
		case api.ToggleMultitap:
			u.Println("Received multitap request from a browser -> relay to worker")
			u.HandleToggleMultitap()
		}
	}
}
