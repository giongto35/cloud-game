package comm

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/goccy/go-json"
)

const callTimeout = 5 * time.Second

var (
	errConnClosed = errors.New("connection closed")
	errTimeout    = errors.New("timeout")
)

type (
	Connector struct {
		tag string
		wu  *websocket.Upgrader
	}
	Client struct {
		conn     *websocket.WS
		queue    map[network.Uid]*call
		onPacket func(packet In)
		mu       sync.Mutex
	}
	call struct {
		done     chan struct{}
		err      error
		Response In
	}
)

var sentPool = sync.Pool{
	New: func() any {
		return Out{}
	},
}

type Option = func(c *Connector)

func WithOrigin(origin string) Option {
	return func(c *Connector) { c.wu = websocket.NewUpgrader(origin) }
}
func WithTag(tag string) Option { return func(c *Connector) { c.tag = tag } }

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

func (co *Connector) NewClientServer(w http.ResponseWriter, r *http.Request, log *logger.Logger) (*SocketClient, error) {
	ws, err := co.wu.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	conn, err := connect(websocket.NewServerWithConn(ws, log))
	if err != nil {
		return nil, err
	}
	c := New(conn, co.tag, network.NewUid(), log)
	defer log.Info().Msg("Connect")
	return &c, nil
}

func (co *Connector) NewClient(address url.URL, log *logger.Logger) (*Client, error) {
	return connect(websocket.NewClient(address, log))
}

func connect(conn *websocket.WS, err error) (*Client, error) {
	if err != nil {
		return nil, err
	}
	client := &Client{conn: conn, queue: make(map[network.Uid]*call, 1)}
	client.conn.OnMessage = client.handleMessage
	return client, nil
}

func (c *Client) OnPacket(fn func(packet In)) { c.mu.Lock(); c.onPacket = fn; c.mu.Unlock() }

func (c *Client) Listen() { c.mu.Lock(); c.conn.Listen(); c.mu.Unlock() }

func (c *Client) Close() {
	// !to handle error
	c.conn.Close()
	c.drain(errConnClosed)
}

func (c *Client) Call(type_ uint8, payload any) ([]byte, error) {
	// !to expose channel instead of results
	rq := sentPool.Get().(Out)
	rq.Id, rq.T, rq.Payload = network.NewUid(), type_, payload
	r, err := json.Marshal(&rq)
	sentPool.Put(rq)
	if err != nil {
		//delete(c.queue, id)
		return nil, err
	}

	task := &call{done: make(chan struct{})}
	c.mu.Lock()
	c.queue[rq.Id] = task
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
	rq := sentPool.Get().(Out)
	rq.Id, rq.T, rq.Payload = "", type_, pl
	defer sentPool.Put(rq)
	return c.SendPacket(rq)
}
func (c *Client) Route(p In, pl Out) error {
	rq := sentPool.Get().(Out)
	rq.Id, rq.T, rq.Payload = p.Id, p.T, pl.Payload
	defer sentPool.Put(rq)
	return c.SendPacket(rq)
}

func (c *Client) SendPacket(packet Out) error {
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

	if res.Id != network.EmptyUid {
		task := c.pop(res.Id)
		if task != nil {
			task.Response = res
			task.done <- struct{}{}
			return
		}
	}
	c.onPacket(res)
}

func (c *Client) pop(id network.Uid) *call {
	c.mu.Lock()
	defer c.mu.Unlock()
	task := c.queue[id]
	delete(c.queue, id)
	return task
}

func (c *Client) drain(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, task := range c.queue {
		if task.err == nil {
			task.err = err
		}
		task.done <- struct{}{}
	}
}
