package com

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type (
	In struct {
		Id      api.Uid         `json:"id,omitempty"`
		T       api.PT          `json:"t"`
		Payload json.RawMessage `json:"p,omitempty"`
	}
	Out struct {
		Id      api.Uid `json:"id,omitempty"`
		T       api.PT  `json:"t"`
		Payload any     `json:"p,omitempty"`
	}
)

var (
	EmptyPacket = Out{Payload: ""}
	ErrPacket   = Out{Payload: "err"}
	OkPacket    = Out{Payload: "ok"}
)

type (
	NetClient interface {
		Disconnect()
		Id() api.Uid
	}
	RegionalClient interface {
		In(region string) bool
	}
)

type SocketClient struct {
	id   api.Uid
	wire *Client
	Tag  string
	Log  *logger.Logger
}

func Unwrap[T any](data []byte) *T {
	out := new(T)
	if err := json.Unmarshal(data, out); err != nil {
		return nil
	}
	return out
}

func UnwrapChecked[T any](bytes []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	return Unwrap[T](bytes), nil
}

func New(conn *Client, tag string, id api.Uid, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("cid", id.Short()))
	dir := "→"
	if conn.IsServer() {
		dir = "←"
	}
	l.Debug().Str("c", tag).Str("d", dir).Msg("Connect")
	return SocketClient{id: id, wire: conn, Tag: tag, Log: l}
}

func (c *SocketClient) SetId(id api.Uid) { c.id = id }

func (c *SocketClient) OnPacket(fn func(p In) error) {
	logFn := func(p In) {
		c.Log.Debug().Str("c", c.Tag).Str("d", "←").Msgf("%s", p.T)
		if err := fn(p); err != nil {
			c.Log.Error().Err(err).Send()
		}
	}
	c.wire.OnPacket(logFn)
}

// Send makes a blocking call.
func (c *SocketClient) Send(t api.PT, data any) ([]byte, error) {
	c.Log.Debug().Str("c", c.Tag).Str("d", "→").Msgf("ᵇ%s", t)
	return c.wire.Call(t, data)
}

// Notify just sends a message and goes further.
func (c *SocketClient) Notify(t api.PT, data any) {
	c.Log.Debug().Str("c", c.Tag).Str("d", "→").Msgf("%s", t)
	_ = c.wire.Send(t, data)
}

func (c *SocketClient) Disconnect() {
	c.wire.Close()
	c.Log.Debug().Str("c", c.Tag).Str("d", "x").Msg("Close")
}

func (c *SocketClient) Id() api.Uid          { return c.id }
func (c *SocketClient) Listen()              { c.ProcessMessages(); <-c.Done() }
func (c *SocketClient) ProcessMessages()     { c.wire.Listen() }
func (c *SocketClient) Route(in In, out Out) { _ = c.wire.Route(in, out) }
func (c *SocketClient) String() string       { return c.Tag + ":" + c.Id().String() }
func (c *SocketClient) Done() chan struct{}  { return c.wire.Wait() }
