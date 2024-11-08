package websocket

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 10 * 1024
	pingTime       = pongTime * 9 / 10
	pongTime       = 7 * time.Second
	writeWait      = 5 * time.Second
)

type Client struct {
	Dialer *websocket.Dialer
}

type Server struct {
	Upgrader *Upgrader
}

type Connection struct {
	alive        bool
	callback     MessageHandler
	conn         deadlineConn
	done         chan struct{}
	errorHandler ErrorHandler
	once         sync.Once
	pingPong     bool
	send         chan []byte
	messSize     int64
}

type deadlineConn struct {
	*websocket.Conn
	wt time.Duration
	mu sync.Mutex // needed for concurrent writes of Gorilla
}

type MessageHandler func([]byte, error)
type ErrorHandler func(err error)

type Upgrader struct {
	websocket.Upgrader
	Origin string
}

var DefaultDialer = websocket.DefaultDialer
var DefaultUpgrader = Upgrader{Upgrader: websocket.Upgrader{
	ReadBufferSize:    2048,
	WriteBufferSize:   2048,
	WriteBufferPool:   &sync.Pool{},
	EnableCompression: true,
}}

func NewUpgrader(origin string) *Upgrader {
	u := DefaultUpgrader
	switch {
	case origin == "*":
		u.CheckOrigin = func(r *http.Request) bool { return true }
	case origin != "":
		u.CheckOrigin = func(r *http.Request) bool { return r.Header.Get("Origin") == origin }
	}
	return &u
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	if u.Origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", u.Origin)
	}
	return u.Upgrader.Upgrade(w, r, responseHeader)
}

func (s *Server) Connect(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Connection, error) {
	u := s.Upgrader
	if u == nil {
		u = &DefaultUpgrader
	}
	conn, err := u.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}
	return newSocket(conn, true), nil
}

func (c *Client) Connect(address url.URL) (*Connection, error) {
	dialer := c.Dialer
	if dialer == nil {
		dialer = DefaultDialer
	}
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, _, err := dialer.Dial(address.String(), nil)
	if err != nil {
		return nil, err
	}
	return newSocket(conn, false), nil
}

func (conn *deadlineConn) write(t int, mess []byte) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(conn.wt)); err != nil {
		return err
	}
	return conn.WriteMessage(t, mess)
}

func (conn *deadlineConn) writeControl(messageType int, data []byte, deadline time.Time) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.Conn.WriteControl(messageType, data, deadline)
}

// reader pumps messages from the websocket connection to the SetMessageHandler callback.
// Blocking, must be called as goroutine. Serializes all websocket reads.
func (c *Connection) reader() {
	defer func() {
		close(c.send)
		c.close()
	}()

	var s int64 = maxMessageSize
	if c.messSize > 0 {
		s = c.messSize
	}
	c.conn.SetReadLimit(s)

	_ = c.conn.SetReadDeadline(time.Now().Add(pongTime))
	if c.pingPong {
		c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
	} else {
		c.conn.SetPingHandler(func(string) error {
			_ = c.conn.SetReadDeadline(time.Now().Add(pongTime))
			err := c.conn.writeControl(websocket.PongMessage, nil, time.Now().Add(writeWait))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				return nil
			}
			return err
		})
	}
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.errorHandler(err)
			} else {
				c.callback(message, err)
			}
			break
		}
		c.callback(message, err)
	}
}

// writer pumps messages from the send channel to the websocket connection.
// Blocking, must be called as goroutine. Serializes all websocket writes.
func (c *Connection) writer() {
	defer c.close()

	if c.pingPong {
		ticker := time.NewTicker(pingTime)
		defer ticker.Stop()

		for {
			select {
			case message, ok := <-c.send:
				if !c.handleMessage(message, ok) {
					return
				}
			case <-ticker.C:
				if err := c.conn.write(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	} else {
		for message := range c.send {
			if !c.handleMessage(message, true) {
				return
			}
		}
	}
}

func (c *Connection) handleMessage(message []byte, ok bool) bool {
	if !ok {
		_ = c.conn.write(websocket.CloseMessage, nil)
		return false
	}
	if err := c.conn.write(websocket.TextMessage, message); err != nil {
		return false
	}
	return true
}

func (c *Connection) close() {
	c.once.Do(func() {
		c.alive = false
		_ = c.conn.Close()
		close(c.done)
	})
}

func newSocket(conn *websocket.Conn, pingPong bool) *Connection {
	return &Connection{
		callback: func(message []byte, err error) {},
		conn:     deadlineConn{Conn: conn, wt: writeWait},
		done:     make(chan struct{}, 1),
		once:     sync.Once{},
		pingPong: pingPong,
		send:     make(chan []byte),
	}
}

// IsServer returns true if the connection has server capabilities and not just a client.
// For now, we assume every connection with ping/pong handler is a server.
func (c *Connection) IsServer() bool { return c.pingPong }

func (c *Connection) SetMessageHandler(fn MessageHandler) { c.callback = fn }

func (c *Connection) SetErrorHandler(fn ErrorHandler) { c.errorHandler = fn }

func (c *Connection) SetMaxMessageSize(s int64) { c.messSize = s }

func (c *Connection) Listen() chan struct{} {
	if c.alive {
		return c.done
	}
	c.alive = true
	go c.writer()
	go c.reader()
	return c.done
}

func (c *Connection) Write(data []byte) {
	if c.alive {
		c.send <- data
	}
}

func (c *Connection) Close() {
	if c.alive {
		_ = c.conn.write(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
}
