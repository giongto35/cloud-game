package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
)

const callTimeout = 5 * time.Second

var (
	errConnClosed = errors.New("connection closed")
	errTimeout    = errors.New("timeout")
)

type call struct {
	done     chan struct{}
	err      error
	Request  OutPacket
	Response InPacket
}

type Client struct {
	Conn *websocket.WS
	// !to check leaks
	queue    map[network.Uid]*call
	mu       sync.Mutex
	onPacket func(packet InPacket)
}

func NewClient(address url.URL, log *logger.Logger) (*Client, error) {
	return connect(websocket.NewClient(address, log))
}

func NewClientServer(w http.ResponseWriter, r *http.Request, u *websocket.Upgrader, log *logger.Logger) (*Client, error) {
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return connect(websocket.NewServerWithConn(conn, log))
}

func connect(conn *websocket.WS, err error) (*Client, error) {
	if err != nil {
		return nil, err
	}
	client := &Client{Conn: conn, queue: make(map[network.Uid]*call, 1)}
	client.Conn.OnMessage = client.handleMessage
	return client, nil
}

func (c *Client) OnPacket(fn func(packet InPacket)) { c.mu.Lock(); c.onPacket = fn; c.mu.Unlock() }

func (c *Client) Listen() { c.mu.Lock(); c.Conn.Listen(); c.mu.Unlock() }

// !to handle error
func (c *Client) Close() {
	c.Conn.Close()
	c.releaseQueue(errConnClosed)
}

// !to expose channel instead of results
func (c *Client) Call(type_ uint8, payload interface{}) ([]byte, error) {
	id := network.NewUid()
	rq := OutPacket{Id: id, T: type_, Payload: payload}
	call := &call{Request: rq, done: make(chan struct{})}
	r, err := json.Marshal(&rq)
	if err != nil {
		delete(c.queue, id)
		return nil, err
	}

	c.mu.Lock()
	c.queue[id] = call
	c.Conn.Write(r)
	c.mu.Unlock()
	select {
	case <-call.done:
	case <-time.After(callTimeout):
		call.err = errTimeout
	}
	return call.Response.Payload, call.err
}

func (c *Client) Send(type_ uint8, payload interface{}) error {
	return c.SendPacket(OutPacket{T: type_, Payload: payload})
}

func (c *Client) SendPacket(packet OutPacket) error {
	r, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.Conn.Write(r)
	c.mu.Unlock()
	return nil
}

func (c *Client) handleMessage(message []byte, err error) {
	if err != nil {
		return
	}

	var res InPacket
	if err = json.Unmarshal(message, &res); err != nil {
		return
	}

	if res.Id != network.EmptyUid {
		call := c.pop(res.Id)
		if call != nil {
			call.Response = res
			call.done <- struct{}{}
			return
		}
	}
	c.onPacket(res)
}

func (c *Client) pop(id network.Uid) *call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.queue[id]
	delete(c.queue, id)
	return call
}

func (c *Client) releaseQueue(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, call := range c.queue {
		call.err = err
		call.done <- struct{}{}
	}
}

func (c *Client) GetRemoteAddr() net.Addr { return c.Conn.GetRemoteAddr() }
