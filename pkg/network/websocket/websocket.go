package websocket

import (
	"crypto/tls"
	"errors"
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
	pongTime       = 5 * time.Second
	writeWait      = 1 * time.Second
)

type (
	WS struct {
		alive    bool
		callback func(message []byte, err error)
		conn     deadlineConn
		done     chan struct{}
		once     sync.Once
		pingPong bool
		send     chan []byte
	}
	Upgrader struct {
		websocket.Upgrader
		origin string
	}
	deadlineConn struct {
		*websocket.Conn
		wt time.Duration
		mu sync.Mutex // needed for concurrent writes of Gorilla
	}
)

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

var DefaultUpgrader = Upgrader{
	Upgrader: websocket.Upgrader{
		ReadBufferSize:    2048,
		WriteBufferSize:   2048,
		WriteBufferPool:   &sync.Pool{},
		EnableCompression: true,
	},
}

var ErrNilConnection = errors.New("nil connection")

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
	if u.origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", u.origin)
	}
	return u.Upgrader.Upgrade(w, r, responseHeader)
}

func NewServerWithConn(conn *websocket.Conn) (*WS, error) {
	if conn == nil {
		return nil, ErrNilConnection
	}
	return newSocket(conn, true), nil
}

func NewClient(address url.URL) (*WS, error) {
	dialer := websocket.DefaultDialer
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, _, err := dialer.Dial(address.String(), nil)
	if err != nil {
		return nil, err
	}
	return newSocket(conn, false), nil
}

// reader pumps messages from the websocket connection to the SetMessageHandler callback.
// Blocking, must be called as goroutine. Serializes all websocket reads.
func (ws *WS) reader() {
	defer func() {
		close(ws.send)
		ws.close()
	}()

	ws.conn.SetReadLimit(maxMessageSize)
	_ = ws.conn.SetReadDeadline(time.Now().Add(pongTime))
	if ws.pingPong {
		ws.conn.SetPongHandler(func(string) error { _ = ws.conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
	} else {
		ws.conn.SetPingHandler(func(string) error {
			_ = ws.conn.SetReadDeadline(time.Now().Add(pongTime))
			err := ws.conn.writeControl(websocket.PongMessage, nil, time.Now().Add(writeWait))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				return nil
			}
			return err
		})
	}
	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				ws.callback(message, err)
			}
			break
		}
		ws.callback(message, err)
	}
}

// writer pumps messages from the send channel to the websocket connection.
// Blocking, must be called as goroutine. Serializes all websocket writes.
func (ws *WS) writer() {
	defer ws.close()

	if ws.pingPong {
		ticker := time.NewTicker(pingTime)
		defer ticker.Stop()

		for {
			select {
			case message, ok := <-ws.send:
				if !ws.handleMessage(message, ok) {
					return
				}
			case <-ticker.C:
				if err := ws.conn.write(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	} else {
		for message := range ws.send {
			if !ws.handleMessage(message, true) {
				return
			}
		}
	}
}

func (ws *WS) handleMessage(message []byte, ok bool) bool {
	if !ok {
		_ = ws.conn.write(websocket.CloseMessage, nil)
		return false
	}
	if err := ws.conn.write(websocket.TextMessage, message); err != nil {
		return false
	}
	return true
}

func (ws *WS) close() {
	ws.once.Do(func() {
		ws.alive = false
		_ = ws.conn.Close()
		close(ws.done)
	})
}

func newSocket(conn *websocket.Conn, pingPong bool) *WS {
	return &WS{
		callback: func(message []byte, err error) {},
		conn:     deadlineConn{Conn: conn, wt: writeWait},
		done:     make(chan struct{}, 1),
		once:     sync.Once{},
		pingPong: pingPong,
		send:     make(chan []byte),
	}
}

func (ws *WS) SetMessageHandler(fn func([]byte, error)) { ws.callback = fn }

func (ws *WS) Listen() chan struct{} {
	ws.alive = true
	go ws.writer()
	go ws.reader()
	return ws.done
}

func (ws *WS) Write(data []byte) {
	if ws.alive {
		ws.send <- data
	}
}

func (ws *WS) Close() {
	if ws.alive {
		_ = ws.conn.write(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
}
