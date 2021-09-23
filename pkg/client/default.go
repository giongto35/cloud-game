package client

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type DefaultClient struct {
	NetClient

	id   network.Uid
	tag  string
	wire *ipc.Client
}

func New(conn *ipc.Client, tag string) DefaultClient {
	return DefaultClient{id: network.NewUid(), wire: conn, tag: tag}
}

func (c DefaultClient) Id() network.Uid { return c.id }

func (c DefaultClient) Send(t uint8, data interface{}) ([]byte, error) {
	return c.wire.Call(t, data)
}

func (c DefaultClient) SendPacket(packet ipc.OutPacket) error {
	return c.wire.SendPacket(packet)
}

func (c DefaultClient) SendAndForget(t uint8, data interface{}) error {
	return c.wire.Send(t, data)
}

func (c DefaultClient) OnPacket(fn func(packet ipc.InPacket)) {
	c.wire.OnPacket = fn
}

func (c DefaultClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("[%s:%s] %s", c.tag, c.id.Short(), format), args...)
}

func (c DefaultClient) Listen() { <-c.wire.Conn.Done }

func (c DefaultClient) Close() { c.wire.Close() }

func (c DefaultClient) String() string { return c.tag + ":" + c.Id().Short() }
