package comm

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type (
	In struct {
		Id      network.Uid     `json:"id,omitempty"`
		T       Type            `json:"t"`
		Payload json.RawMessage `json:"p,omitempty"`
	}
	Out struct {
		Id      network.Uid `json:"id,omitempty"`
		T       Type        `json:"t"`
		Payload any         `json:"p,omitempty"`
	}
	Type = uint8
)

var (
	EmptyPacket = Out{Payload: ""}
	ErrPacket   = Out{Payload: "err"}
	OkPacket    = Out{Payload: "ok"}
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

func New(conn *Client, tag string, id network.Uid, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("c-uid", string(id)).Str("c-tag", tag))
	return SocketClient{id: id, wire: conn, tag: tag, Log: l}
}

func (c SocketClient) Close()                                { c.wire.Close() }
func (c SocketClient) GetLogger() *logger.Logger             { return c.Log }
func (c SocketClient) Id() network.Uid                       { return c.id }
func (c SocketClient) Listen()                               { c.ProcessMessages(); c.Wait() }
func (c SocketClient) Notify(t Type, data any)               { _ = c.wire.Send(t, data) }
func (c SocketClient) OnPacket(fn func(In))                  { c.wire.OnPacket(fn) }
func (c SocketClient) ProcessMessages()                      { c.wire.Listen() }
func (c SocketClient) Route(p In, pl Out)                    { _ = c.wire.Route(p, pl) }
func (c SocketClient) Send(t Type, data any) ([]byte, error) { return c.wire.Call(t, data) }
func (c SocketClient) String() string                        { return c.tag + ":" + string(c.Id()) }
func (c SocketClient) Wait()                                 { <-c.wire.Wait() }
