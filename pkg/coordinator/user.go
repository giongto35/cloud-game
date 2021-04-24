package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type User struct {
	id     network.Uid
	RoomID string
	wire   *ipc.Client
	Worker *WorkerClient
}

func NewUser(conn *ipc.Client) *User {

	user := &User{
		id: network.NewUid(),
	}

	user.wire = conn

	return user
}

func (u *User) AssignWorker(w *WorkerClient) {
	u.Worker = w
	w.makeAvailable(false)
}

func (u *User) RetainWorker() {
	if u.Worker != nil {
		u.Worker.makeAvailable(true)
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
	log.Printf(fmt.Sprintf("user: [%s] %s", u.id.Short(), format), args...)
}

func (u *User) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("user: [%s] %s", u.id.Short(), fmt.Sprint(args...)))
}
