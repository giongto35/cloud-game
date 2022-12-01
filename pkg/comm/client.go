package comm

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type (
	In struct {
		Id      network.Uid     `json:"id,omitempty"`
		T       api.PT          `json:"t"`
		Payload json.RawMessage `json:"p,omitempty"`
	}
	Out struct {
		Id      network.Uid `json:"id,omitempty"`
		T       api.PT      `json:"t"`
		Payload any         `json:"p,omitempty"`
	}
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
	Tag  string
	wire *Client
	Log  *logger.Logger
}

func New(conn *Client, tag string, id network.Uid, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("cid", id.Short()))
	return SocketClient{id: id, wire: conn, Tag: tag, Log: l}
}
func (c SocketClient) OnPacket(fn func(p In) error) {
	logFn := func(p In) {
		c.Log.Info().Str("c", c.Tag).Str("d", "←").Msgf("%s", p.T)
		if err := fn(p); err != nil {
			c.Log.Error().Err(err).Send()
		}
	}
	c.wire.OnPacket(logFn)
}
func (c SocketClient) Send(t api.PT, data any) ([]byte, error) {
	c.Log.Info().Str("c", c.Tag).Str("d", "→").Msgf("ᵇ%s", t)
	return c.wire.Call(t, data)
}
func (c SocketClient) Notify(t api.PT, data any) {
	c.Log.Info().Str("c", c.Tag).Str("d", "→").Msgf("%s", t)
	_ = c.wire.Send(t, data)
}

func (c SocketClient) Close()               { c.wire.Close() }
func (c SocketClient) Id() network.Uid      { return c.id }
func (c SocketClient) Listen()              { c.ProcessMessages(); c.Wait() }
func (c SocketClient) ProcessMessages()     { c.wire.Listen() }
func (c SocketClient) Route(in In, out Out) { _ = c.wire.Route(in, out) }
func (c SocketClient) String() string       { return c.Tag + ":" + string(c.Id()) }
func (c SocketClient) Wait()                { <-c.wire.Wait() }
