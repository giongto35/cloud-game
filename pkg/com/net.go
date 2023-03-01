package com

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/goccy/go-json"
)

type Id interface {
	Uid
	Generate() Uid
	IsEmpty() bool
	String() string
}

type HasCallId interface {
	SetGetId(fmt.Stringer)
}

type Writer interface {
	Write([]byte)
}

type PacketType interface {
	~uint8
}

type Packet[I Id, T PacketType] interface {
	GetId() I
	GetType() T
	GetPayload() []byte
	HasId() bool
}

type Packet2[T any] interface {
	SetId(string)
	SetType(uint8)
	SetPayload(any)
	SetGetId(fmt.Stringer)
	GetPayload() any
	*T // non-interface type constraint element
}

type Transport[I Id, T PacketType, P Packet[I, T]] struct {
	CallTimeout time.Duration
	Handler     func(P)

	calls Map[I, *request]
}

type request struct {
	done     chan struct{}
	err      error
	response []byte
}

const DefaultCallTimeout = 5 * time.Second

var errCanceled = errors.New("canceled")
var errTimeout = errors.New("timeout")

type (
	Client struct {
		websocket.Client
	}
	Server struct {
		websocket.Server
	}
	Connection struct {
		conn *websocket.Connection
	}
)

func (c *Client) Connect(addr url.URL) (*Connection, error) { return connect(c.Client.Connect(addr)) }

func (s *Server) Origin(host string) { s.Upgrader = websocket.NewUpgrader(host) }

func (s *Server) Connect(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	return connect(s.Server.Connect(w, r, nil))
}

func (c Connection) IsServer() bool { return c.conn.IsServer() }

func connect(conn *websocket.Connection, err error) (*Connection, error) {
	if err != nil {
		return nil, err
	}
	return &Connection{conn: conn}, nil
}

func (t *Transport[_, _, _]) SendAsync(w Writer, packet any) error {
	r, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	w.Write(r)
	return nil
}

func (t *Transport[I, _, _]) SendSync(w Writer, rq HasCallId) ([]byte, error) {
	// generate new uid for the new call
	// by setting it in the incoming rq param
	id := I((*new(I)).Generate())
	rq.SetGetId(id)

	r, err := json.Marshal(rq)
	if err != nil {
		return nil, err
	}
	task := &request{done: make(chan struct{})}
	t.calls.Put(id, task)
	w.Write(r)
	select {
	case <-task.done:
	case <-time.After(t.callTimeout()):
		task.err = errTimeout
	}
	return task.response, task.err
}

func (t *Transport[_, _, P]) handleMessage(message []byte) error {
	res := *new(P)
	if err := json.Unmarshal(message, &res); err != nil {
		return err
	}
	// if we have an id, then unblock blocking call with that id
	if res.HasId() {
		if blocked := t.calls.Pop(res.GetId()); blocked != nil {
			blocked.response = res.GetPayload()
			close(blocked.done)
			return nil
		}
	}
	if t.Handler != nil {
		t.Handler(res)
	}
	return nil
}

func (t *Transport[_, _, _]) callTimeout() time.Duration {
	if t.CallTimeout > 0 {
		return t.CallTimeout
	}
	return DefaultCallTimeout
}

func (t *Transport[_, _, _]) Clean() {
	// drain cancels all what's left in the task queue.
	t.calls.ForEach(func(task *request) {
		if task.err == nil {
			task.err = errCanceled
		}
		close(task.done)
	})
}
