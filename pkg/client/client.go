package client

import (
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type (
	NetClient interface {
		Close()
		Id() network.Uid
	}
	RegionalClient interface {
		In(region string) bool
	}
)

type SocketClient struct {
	NetClient

	id   network.Uid
	tag  string
	wire *Client
	Log  *logger.Logger
}

func New(conn *Client, tag string, log *logger.Logger) SocketClient {
	return NewWithId(network.NewUid(), conn, tag, log)
}

func NewWithId(id network.Uid, conn *Client, tag string, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("c-uid", string(id)).Str("c-tag", tag))
	return SocketClient{id: id, wire: conn, tag: tag, Log: l}
}

func (c SocketClient) Id() network.Uid { return c.id }

func (c SocketClient) Send(t PacketType, data any) ([]byte, error) {
	return c.wire.Call(t, data)
}

func (c SocketClient) SendPacket(packet OutPacket) error { return c.wire.SendPacket(packet) }

func (c SocketClient) SendAndForget(t PacketType, data any) error {
	return c.wire.Send(t, data)
}

// Notify supposedly non-blocking, discard error operation.
func (c SocketClient) Notify(t PacketType, data any) { _ = c.wire.Send(t, data) }

func (c SocketClient) OnPacket(fn func(InPacket)) { c.wire.OnPacket(fn) }

func (c SocketClient) Route(p InPacket, payload any) error {
	return c.wire.SendPacket(p.Proxy(payload))
}

func (c SocketClient) GetLogger() *logger.Logger { return c.Log }

func (c SocketClient) ProcessMessages() { c.wire.Listen() }

func (c SocketClient) Wait() { <-c.wire.Wait() }

func (c SocketClient) Listen() { c.ProcessMessages(); c.Wait() }

func (c SocketClient) Close() { c.wire.Close() }

func (c SocketClient) String() string { return c.tag + ":" + string(c.Id()) }
