package ipc

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	ws "github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/gorilla/websocket"
)

const callTimeout = 5 * time.Second

type Client struct {
	Conn *ws.WS
	// !to check leaks
	queue map[network.Uid]*Call
	mu    sync.Mutex

	OnPacket func(packet Packet)
}

func NewClient(address url.URL) (*Client, error) {
	conn := ws.NewClient(address)
	if conn == nil {
		return nil, errors.New("can't connect")
	}
	client := &Client{
		Conn:  conn,
		queue: make(map[network.Uid]*Call, 1),
	}
	client.Conn.OnMessage = client.handleMessage
	return client, nil
}

func NewClientServer(w http.ResponseWriter, r *http.Request) (*Client, error) {
	conn := ws.NewServer(w, r)
	if conn == nil {
		return nil, errors.New("can't connect")
	}
	client := &Client{
		Conn:  conn,
		queue: make(map[network.Uid]*Call, 1),
	}
	client.Conn.OnMessage = client.handleMessage
	return client, nil
}

func Connect(address url.URL) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, _, err := dialer.Dial(address.String(), nil)
	return conn, err
}

// !to handle error
func (c *Client) Close() {
	c.Conn.Close()
	c.releaseQueue(errors.New("connection closed"))
}

// !to expose channel instead of results
func (c *Client) Call(type_ uint8, payload interface{}) (interface{}, error) {
	id := network.NewUid()
	rq := Packet{Id: id, T: PacketType(type_), Payload: payload}
	call := &Call{Request: rq, done: make(chan struct{})}
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
		call.err = errors.New("timeout")
	}
	return call.Response.Payload, call.err
}

func (c *Client) Send(type_ uint8, payload interface{}) (interface{}, error) {
	rq := Packet{T: PacketType(type_), Payload: payload}
	call := &Call{Request: rq}
	r, err := json.Marshal(&rq)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.Conn.Write(r)
	c.mu.Unlock()

	return call.Response.Payload, call.err
}

func (c *Client) handleMessage(message []byte, err error) {
	if err != nil {
		return
	}

	var res Packet
	_ = json.Unmarshal(message, &res)

	// skip block on "async" calls
	if res.Id == network.EmptyUid {
		c.OnPacket(res)
		return
	}

	call := c.pop(res.Id)
	if call == nil {
		log.Printf("no pending request found")
		return
	}

	call.Response = res
	call.done <- struct{}{}
}

func (c *Client) pop(id network.Uid) *Call {
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
