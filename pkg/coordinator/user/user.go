package user

import (
	"fmt"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/coordinator/worker"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type User struct {
	Id     network.Uid
	RoomID string
	wire   *ipc.Client
	Worker *worker.WorkerClient
}

func New(conn *ipc.Client) User {
	return User{
		Id:   network.NewUid(),
		wire: conn,
	}
}

func (u *User) AssignWorker(w *worker.WorkerClient) {
	u.Printf("Assigned wkr: %v", w.Id)
	u.Worker = w
	w.MakeAvailable(false)
}

func (u *User) RetainWorker() {
	if u.Worker != nil {
		u.Worker.MakeAvailable(true)
	}
}

func (u *User) AssignRoom(id string) {
	u.RoomID = id
}

func (u *User) Send(t uint8, data interface{}) (interface{}, error) {
	return u.wire.Call(t, data)
}

func (u *User) SendAndForget(t uint8, data interface{}) (interface{}, error) {
	return u.wire.Send(t, data)
}

func (u *User) Handle(fn func(p ipc.Packet)) {
	u.wire.OnPacket = fn
}

func (u *User) WaitDisconnect() {
	<-u.wire.Conn.Done
}

func (u *User) Clean() {
	u.wire.Close()
}

func (u *User) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("user: [%s] %s", u.Id.Short(), format), args...)
}

func (u *User) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("user: [%s] %s", u.Id.Short(), fmt.Sprint(args...)))
}

func (u *User) HandleRequests(launcher launcher.Launcher) {
	u.Handle(func(p ipc.Packet) {
		switch p.T {
		case ipc.PacketType(WebrtcInit):
			u.Printf("Received init_webrtc request -> relay to worker: %s", u.Worker)
			// initWebrtc now only sends signal to worker, asks it to createOffer
			// relay request to target worker
			// worker creates a PeerConnection, and createOffer
			// send SDP back to browser
			u.HandleWebrtcInit()
			u.Println("Received SDP from worker -> sending back to browser")
		case ipc.PacketType(WebrtcAnswer):
			u.Println("Received browser answered SDP -> relay to worker")
			u.HandleWebrtcAnswer(p.Payload)
		case ipc.PacketType(WebrtcIceCandidate):
			u.Println("Received IceCandidate from browser -> relay to worker")
			u.HandleWebrtcIceCandidate(p.Payload)
		case ipc.PacketType(StartGame):
			u.Println("Received start request from a browser -> relay to worker")
			u.HandleStartGame(p.Payload, launcher)
		case ipc.PacketType(QuitGame):
			u.Println("Received quit request from a browser -> relay to worker")
			u.HandleQuitGame(p.Payload)
		case ipc.PacketType(SaveGame):
			u.Println("Received save request from a browser -> relay to worker")
			u.HandleSaveGame()
		case ipc.PacketType(LoadGame):
			u.Println("Received load request from a browser -> relay to worker")
			u.HandleLoadGame()
		case ipc.PacketType(ChangePlayer):
			u.Println("Received update player index request from a browser -> relay to worker")
			u.HandleChangePlayer(p.Payload)
		case ipc.PacketType(ToggleMultitap):
			u.Println("Received multitap request from a browser -> relay to worker")
			u.HandleToggleMultitap()
		}
	})
}
