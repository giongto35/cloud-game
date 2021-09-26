package client

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type (
	NetClient interface {
		Close()
		Id() network.Uid
		Logf(format string, args ...interface{})
	}
	RegionalClient interface {
		In(region string) bool
	}
)

type SocketClient struct {
	NetClient

	id   network.Uid
	tag  string
	wire *ipc.Client
}

func New(conn *ipc.Client, tag string) SocketClient {
	return SocketClient{id: network.NewUid(), wire: conn, tag: tag}
}

func (c SocketClient) Id() network.Uid { return c.id }

func (c SocketClient) Send(t uint8, data interface{}) ([]byte, error) {
	return c.wire.Call(t, data)
}

func (c SocketClient) SendPacket(packet ipc.OutPacket) error {
	return c.wire.SendPacket(packet)
}

func (c SocketClient) SendAndForget(t uint8, data interface{}) error {
	return c.wire.Send(t, data)
}

func (c SocketClient) OnPacket(fn func(packet ipc.InPacket)) {
	c.wire.OnPacket = fn
}

func (c SocketClient) Logf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("[%s:%s] %s", c.tag, c.id.Short(), format), args...)
}

func (c SocketClient) Listen() { <-c.wire.Conn.Done }

func (c SocketClient) Close() { c.wire.Close() }

func (c SocketClient) String() string { return c.tag + ":" + c.Id().Short() }
