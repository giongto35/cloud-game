package com

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/goccy/go-json"
)

type (
	Connector struct {
		tag string
		wu  *websocket.Upgrader
	}
	Client struct {
		conn     *websocket.WS
		queue    map[Uid]*call
		onPacket func(packet In)
		mu       sync.Mutex
	}
	call struct {
		done     chan struct{}
		err      error
		Response In
	}
	Option = func(c *Connector)
)

var (
	errConnClosed = errors.New("connection closed")
	errTimeout    = errors.New("timeout")
)
var outPool = sync.Pool{New: func() any { o := Out{}; return &o }}

func WithOrigin(url string) Option { return func(c *Connector) { c.wu = websocket.NewUpgrader(url) } }
func WithTag(tag string) Option    { return func(c *Connector) { c.tag = tag } }

const callTimeout = 5 * time.Second

func NewConnector(opts ...Option) *Connector {
	c := &Connector{}
	for _, opt := range opts {
		opt(c)
	}
	if c.wu == nil {
		c.wu = &websocket.DefaultUpgrader
	}
	return c
}

func (co *Connector) NewServer(w http.ResponseWriter, r *http.Request, log *logger.Logger) (*SocketClient, error) {
	ws, err := co.wu.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	conn, err := connect(websocket.NewServerWithConn(ws, log))
	if err != nil {
		return nil, err
	}
	c := New(conn, co.tag, NewUid(), log)
	return &c, nil
}

func (co *Connector) NewClient(address url.URL, log *logger.Logger) (*Client, error) {
	return connect(websocket.NewClient(address, log))
}

func connect(conn *websocket.WS, err error) (*Client, error) {
	if err != nil {
		return nil, err
	}
	client := &Client{conn: conn, queue: make(map[Uid]*call, 1)}
	client.conn.OnMessage = client.handleMessage
	return client, nil
}

func (c *Client) IsServer() bool { return c.conn.IsServer() }

func (c *Client) OnPacket(fn func(packet In)) { c.mu.Lock(); c.onPacket = fn; c.mu.Unlock() }

func (c *Client) Listen() { c.mu.Lock(); c.conn.Listen(); c.mu.Unlock() }

func (c *Client) Close() {
	// !to handle error
	c.conn.Close()
	c.drain(errConnClosed)
}

func (c *Client) Call(type_ uint8, payload any) ([]byte, error) {
	// !to expose channel instead of results
	rq := outPool.Get().(*Out)
	id := NewUid()
	rq.Id, rq.T, rq.Payload = id.String(), type_, payload
	r, err := json.Marshal(rq)
	outPool.Put(rq)
	if err != nil {
		//delete(c.queue, id)
		return nil, err
	}

	task := &call{done: make(chan struct{})}
	c.mu.Lock()
	c.queue[id] = task
	c.conn.Write(r)
	c.mu.Unlock()
	select {
	case <-task.done:
	case <-time.After(callTimeout):
		task.err = errTimeout
	}
	return task.Response.Payload, task.err
}

func (c *Client) Send(type_ uint8, pl any) error {
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = "", type_, pl
	defer outPool.Put(rq)
	return c.SendPacket(rq)
}

func (c *Client) Route(p In, pl Out) error {
	rq := outPool.Get().(*Out)
	rq.Id, rq.T, rq.Payload = p.Id.String(), uint8(p.T), pl.Payload
	defer outPool.Put(rq)
	return c.SendPacket(rq)
}

func (c *Client) SendPacket(packet *Out) error {
	r, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn.Write(r)
	c.mu.Unlock()
	return nil
}

func (c *Client) Wait() chan struct{} { return c.conn.Done }

func (c *Client) handleMessage(message []byte, err error) {
	if err != nil {
		return
	}

	var res In
	if err = json.Unmarshal(message, &res); err != nil {
		return
	}

	// empty id implies that we won't track (wait) the response
	if !res.Id.IsEmpty() {
		if task := c.pop(res.Id); task != nil {
			task.Response = res
			close(task.done)
			return
		}
	}
	c.onPacket(res)
}

// pop extracts and removes a task from the queue by its id.
func (c *Client) pop(id Uid) *call {
	c.mu.Lock()
	task := c.queue[id]
	delete(c.queue, id)
	c.mu.Unlock()
	return task
}

// drain cancels all what's left in the task queue.
func (c *Client) drain(err error) {
	c.mu.Lock()
	for _, task := range c.queue {
		if task.err == nil {
			task.err = err
		}
		close(task.done)
	}
	c.mu.Unlock()
}
