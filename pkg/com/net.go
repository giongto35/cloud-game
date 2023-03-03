package com

import (
	"errors"
	"fmt"
	"github.com/rs/xid"
	"net/http"
	"net/url"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/goccy/go-json"
)

type Uid struct {
	xid.ID
}

var NilUid = Uid{xid.NilID()}

func NewUid() Uid { return Uid{xid.New()} }

func (u Uid) Short() string { return u.String()[:3] + "." + u.String()[len(u.String())-3:] }

type HasCallId interface {
	SetGetId(fmt.Stringer)
}

type Writer interface {
	Write([]byte)
}

type Packet[T ~uint8] interface {
	GetId() Uid
	GetType() T
	GetPayload() []byte
}

type Packet2[T any] interface {
	SetId(string)
	SetType(uint8)
	SetPayload(any)
	SetGetId(fmt.Stringer)
	GetPayload() any
	*T // non-interface type constraint element
}

type RPC[T ~uint8, P Packet[T]] struct {
	CallTimeout time.Duration
	Handler     func(P)

	calls Map[Uid, *request]
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

func (t *RPC[_, _]) Send(w Writer, packet any) error {
	r, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	w.Write(r)
	return nil
}

func (t *RPC[_, _]) Call(w Writer, rq HasCallId) ([]byte, error) {
	// generate new uid for the new call
	// by setting it in the incoming rq param
	id := NewUid()
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

func (t *RPC[_, P]) handleMessage(message []byte) error {
	res := *new(P)
	if err := json.Unmarshal(message, &res); err != nil {
		return err
	}
	// if we have an id, then unblock blocking call with that id
	id := res.GetId()
	if id != NilUid {
		if blocked := t.calls.Pop(id); blocked != nil {
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

func (t *RPC[_, _]) callTimeout() time.Duration {
	if t.CallTimeout > 0 {
		return t.CallTimeout
	}
	return DefaultCallTimeout
}

func (t *RPC[_, _]) Cleanup() {
	// drain cancels all what's left in the task queue.
	t.calls.ForEach(func(task *request) {
		if task.err == nil {
			task.err = errCanceled
		}
		close(task.done)
	})
}
