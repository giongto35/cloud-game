package com

import (
	"encoding/base64"
	"sync"

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

func (o *Out) SetId(id Uid) { o.Id = id.String() }

var (
	EmptyPacket = Out{Payload: ""}
	ErrPacket   = Out{Payload: "err"}
	OkPacket    = Out{Payload: "ok"}
)

var outPool = sync.Pool{New: func() any { o := Out{}; return &o }}

type (
	NetClient[K comparable] interface {
		Disconnect()
		Id() K
	}
	RegionalClient interface {
		In(region string) bool
	}
)

type SocketConnector struct{}

type SocketClient struct {
	id     Uid
	client *Client
	Log    *logger.Logger
	log    *logger.Logger // special logger with x -> y directions
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

func NewConnection(conn *Client, id Uid, isServer bool, tag string, log *logger.Logger) *SocketClient {
	if id.IsNil() {
		id = NewUid()
	}
	extLog := log.Extend(log.With().Str("cid", id.Short()))
	dir := "→"
	if isServer {
		dir = "←"
	}
	intLog := extLog.Extend(
		extLog.With().
			Str(logger.ClientField, tag).
			Str(logger.DirectionField, dir),
	)

	intLog.Debug().Msg("Connect")
	return &SocketClient{id: id, client: conn, log: intLog, Log: extLog}
}

func (c *SocketClient) OnPacket(fn func(p In) error) {
	logFn := func(p In) {
		c.log.Debug().Str(logger.DirectionField, "←").Msgf("%s", p.T)
		if err := fn(p); err != nil {
			c.Log.Error().Err(err).Send()
		}
	}
	c.client.OnPacket(logFn)
}

func (c *SocketClient) Route(in In, out Out) {
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = in.Id.String(), uint8(in.T), out.Payload
	defer outPool.Put(rq)
	_ = c.client.SendPacket(rq)
}

// Send makes a blocking call.
func (c *SocketClient) Send(t api.PT, data any) ([]byte, error) {
	c.log.Debug().Str(logger.DirectionField, "→").Msgf("ᵇ%s", t)
	rq := outPool.Get().(*Out)
	rq.T, rq.Payload = uint8(t), data
	defer outPool.Put(rq)
	return c.client.Call(rq)
}

// Notify just sends a message and goes further.
func (c *SocketClient) Notify(t api.PT, data any) {
	c.log.Debug().Str(logger.DirectionField, "→").Msgf("%s", t)
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = "", uint8(t), data
	defer outPool.Put(rq)
	_ = c.client.SendPacket(rq)
}

func (c *SocketClient) Disconnect() {
	c.client.Close()
	c.log.Debug().Str(logger.DirectionField, "x").Msg("Close")
}

func (c *SocketClient) Id() Uid               { return c.id }
func (c *SocketClient) Listen() chan struct{} { return c.client.Listen() }
func (c *SocketClient) String() string        { return c.Id().String() }

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
