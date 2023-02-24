package com

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/goccy/go-json"
)

type (
	ClientConnector struct {
		websocket.Client
	}
	ServerConnector struct {
		websocket.Server
	}
	Client struct {
		conn     *websocket.Connection
		queue    map[Uid]*call
		onPacket func(packet In)
		mu       sync.Mutex
	}
	call struct {
		done     chan struct{}
		err      error
		response []byte
	}
	HasCallId interface {
		SetId(Uid)
	}
)

var (
	errConnClosed = errors.New("connection closed")
	errTimeout    = errors.New("timeout")
)

const callTimeout = 5 * time.Second

func (c *ClientConnector) Connect(address url.URL) (*Client, error) {
	return connect(c.Client.Connect(address))
}

func (s *ServerConnector) Origin(host string) { s.Upgrader = websocket.NewUpgrader(host) }

func (s *ServerConnector) Connect(w http.ResponseWriter, r *http.Request) (*Client, error) {
	return connect(s.Server.Connect(w, r, nil))
}

func connect(conn *websocket.Connection, err error) (*Client, error) {
	if err != nil {
		return nil, err
	}
	client := &Client{conn: conn, queue: make(map[Uid]*call, 1)}
	client.conn.SetMessageHandler(client.handleMessage)
	return client, nil
}

func (c *Client) OnPacket(fn func(packet In)) {
	c.mu.Lock()
	c.onPacket = fn
	c.mu.Unlock()
}

func (c *Client) Listen() chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Listen()
}

func (c *Client) Close() {
	// !to handle error
	c.conn.Close()
	c.drain(errConnClosed)
}

func (c *Client) Call(rq HasCallId) ([]byte, error) {
	id := NewUid()
	rq.SetId(id)
	// !to expose channel instead of results
	r, err := json.Marshal(rq)
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
	return task.response, task.err
}

func (c *Client) SendPacket(packet any) error {
	r, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn.Write(r)
	c.mu.Unlock()
	return nil
}

func (c *Client) handleMessage(message []byte, err error) {
	if err != nil {
		return
	}

	var res In
	if err = json.Unmarshal(message, &res); err != nil {
		return
	}

	// if we have an id, then unblock blocking call with that id
	if !res.Id.IsEmpty() {
		c.mu.Lock()
		blocked := c.queue[res.Id]
		delete(c.queue, res.Id)
		c.mu.Unlock()
		if blocked != nil {
			blocked.response = res.Payload
			close(blocked.done)
			return
		}
	}
	c.onPacket(res)
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
