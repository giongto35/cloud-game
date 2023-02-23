package com

import (
	"encoding/base64"
	"net/http"
	"net/url"
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

type SocketConnector struct {
	connector *Connector
}

type SocketClient struct {
	id   Uid
	wire *Client
	Tag  string
	Log  *logger.Logger
}

type Options struct {
	Id      Uid
	Address url.URL
	R       *http.Request
	W       http.ResponseWriter
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

func NewSocketConnector(opts ...Option) *SocketConnector {
	return &SocketConnector{NewConnector(opts...)}
}

func (sc *SocketConnector) NewConnection(opts Options, log *logger.Logger) (*SocketClient, error) {
	id := opts.Id
	if id.IsNil() {
		id = NewUid()
	}
	l := log.Extend(log.With().Str("cid", id.Short()))
	dir := "→"
	if sc.connector.isServer {
		dir = "←"
	}

	l.Debug().Str("c", sc.connector.tag).Str("d", dir).Msg("Connect")
	var conn *Client
	var err error
	if sc.connector.isServer {
		conn, err = sc.connector.NewServer(opts.W, opts.R)
	} else {
		conn, err = sc.connector.NewClient(opts.Address)
	}
	if err != nil {
		return nil, err
	}
	return &SocketClient{id: id, wire: conn, Tag: sc.connector.tag, Log: l}, nil
}

func (c *SocketClient) OnPacket(fn func(p In) error) {
	logFn := func(p In) {
		c.Log.Debug().Str("c", c.Tag).Str("d", "←").Msgf("%s", p.T)
		if err := fn(p); err != nil {
			c.Log.Error().Err(err).Send()
		}
	}
	c.wire.OnPacket(logFn)
}

func (c *SocketClient) Route(in In, out Out) {
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = in.Id.String(), uint8(in.T), out.Payload
	defer outPool.Put(rq)
	_ = c.wire.SendPacket(rq)
}

// Send makes a blocking call.
func (c *SocketClient) Send(t api.PT, data any) ([]byte, error) {
	c.Log.Debug().Str("c", c.Tag).Str("d", "→").Msgf("ᵇ%s", t)
	rq := outPool.Get().(*Out)
	rq.T, rq.Payload = uint8(t), data
	defer outPool.Put(rq)
	return c.wire.Call(rq)
}

// Notify just sends a message and goes further.
func (c *SocketClient) Notify(t api.PT, data any) {
	c.Log.Debug().Str("c", c.Tag).Str("d", "→").Msgf("%s", t)
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = "", uint8(t), data
	defer outPool.Put(rq)
	_ = c.wire.SendPacket(rq)
}

func (c *SocketClient) Disconnect() {
	c.wire.Close()
	c.Log.Debug().Str("c", c.Tag).Str("d", "x").Msg("Close")
}

func (c *SocketClient) Id() Uid               { return c.id }
func (c *SocketClient) Listen() chan struct{} { return c.wire.Listen() }
func (c *SocketClient) String() string        { return c.Tag + ":" + c.Id().String() }

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
