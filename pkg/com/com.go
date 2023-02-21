package com

import (
	"encoding/base64"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/goccy/go-json"
)

type (
	In struct {
		Id      Uid             `json:"id,omitempty"`
		T       api.PT          `json:"t"`
		Payload json.RawMessage `json:"p,omitempty"`
	}
	Out struct {
		Id      string `json:"id,omitempty"`
		T       uint8  `json:"t"`
		Payload any    `json:"p,omitempty"`
	}
)

var (
	EmptyPacket = Out{Payload: ""}
	ErrPacket   = Out{Payload: "err"}
	OkPacket    = Out{Payload: "ok"}
)

type (
	NetClient[K comparable] interface {
		Disconnect()
		Id() K
	}
	RegionalClient interface {
		In(region string) bool
	}
)

type SocketClient struct {
	id   Uid
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

func New(conn *Client, tag string, id Uid, log *logger.Logger) SocketClient {
	l := log.Extend(log.With().Str("cid", id.Short()))
	dir := "→"
	if conn.IsServer() {
		dir = "←"
	}
	l.Debug().Str("c", tag).Str("d", dir).Msg("Connect")
	return SocketClient{id: id, wire: conn, Tag: tag, Log: l}
}

func (c *SocketClient) SetId(id Uid) { c.id = id }

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
	return c.wire.Call(uint8(t), data)
}

// Notify just sends a message and goes further.
func (c *SocketClient) Notify(t api.PT, data any) {
	c.Log.Debug().Str("c", c.Tag).Str("d", "→").Msgf("%s", t)
	_ = c.wire.Send(uint8(t), data)
}

func (c *SocketClient) Disconnect() {
	c.wire.Close()
	c.Log.Debug().Str("c", c.Tag).Str("d", "x").Msg("Close")
}

func (c *SocketClient) Id() Uid              { return c.id }
func (c *SocketClient) Listen()              { c.ProcessMessages(); <-c.Done() }
func (c *SocketClient) ProcessMessages()     { c.wire.Listen() }
func (c *SocketClient) Route(in In, out Out) { _ = c.wire.Route(in, out) }
func (c *SocketClient) String() string       { return c.Tag + ":" + c.Id().String() }
func (c *SocketClient) Done() chan struct{}  { return c.wire.Wait() }

// ToBase64Json encodes data to a URL-encoded Base64+JSON string.
func ToBase64Json(data any) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// FromBase64Json decodes data from a URL-encoded Base64+JSON string.
func FromBase64Json(data string, obj any) error {
	b, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
}
