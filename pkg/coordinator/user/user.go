package user

import (
	"fmt"
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
	u.Worker = w
	w.MakeAvailable(false)
}

func (u *User) RetainWorker() {
	if u.Worker != nil {
		u.Worker.MakeAvailable(true)
	}
}

func (u *User) Send(t uint8, data interface{}) (interface{}, error) {
	return u.wire.Call(t, ipc.Payload(data))
}

func (u *User) SendAndForget(t uint8, data interface{}) (interface{}, error) {
	return u.wire.Send(t, ipc.Payload(data))
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
