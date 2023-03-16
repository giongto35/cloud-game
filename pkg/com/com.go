package com

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/network"
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
	wire *Client
	Tag  string
	Log  *logger.Logger
}

func New(conn *Client, tag string, id network.Uid, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("cid", id.Short()))
	dir := "→"
	if conn.IsServer() {
		dir = "←"
	}
	l.Debug().Str("c", tag).Str("d", dir).Msg("Connect")
	return SocketClient{id: id, wire: conn, Tag: tag, Log: l}
}

func (c *SocketClient) SetId(id network.Uid) { c.id = id }

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

func (c *SocketClient) Close() {
	c.wire.Close()
	c.Log.Debug().Str("c", c.Tag).Str("d", "x").Msg("Close")
}

func (c *SocketClient) Id() network.Uid      { return c.id }
func (c *SocketClient) Listen()              { c.ProcessMessages(); <-c.Done() }
func (c *SocketClient) ProcessMessages()     { c.wire.Listen() }
func (c *SocketClient) Route(in In, out Out) { _ = c.wire.Route(in, out) }
func (c *SocketClient) String() string       { return c.Tag + ":" + string(c.Id()) }
func (c *SocketClient) Done() chan struct{}  { return c.wire.Wait() }
