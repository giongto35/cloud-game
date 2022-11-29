package websocket

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
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
		conn      deadlineConn
		send      chan []byte
		OnMessage WSMessageHandler
		pingPong  bool
		once      sync.Once
		Done      chan struct{}
		closed    bool
		log       *logger.Logger
	}
	WSMessageHandler func(message []byte, err error)
	Upgrader         struct {
		websocket.Upgrader
		origin string
	}
	deadlineConn struct {
		*websocket.Conn
		wt time.Duration
	}
)

func (conn *deadlineConn) write(t int, mess []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(conn.wt)); err != nil {
		return err
	}
	return conn.WriteMessage(t, mess)
}

var DefaultUpgrader = Upgrader{
	Upgrader: websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
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

func NewServerWithConn(conn *websocket.Conn, log *logger.Logger) (*WS, error) {
	if conn == nil {
		return nil, ErrNilConnection
	}
	return newSocket(conn, true, log), nil
}

func NewClient(address url.URL, log *logger.Logger) (*WS, error) {
	dialer := websocket.DefaultDialer
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, _, err := dialer.Dial(address.String(), nil)
	if err != nil {
		return nil, err
	}
	return newSocket(conn, false, log), nil
}

// reader pumps messages from the websocket connection to the OnMessage callback.
// Blocking, must be called as goroutine. Serializes all websocket reads.
func (ws *WS) reader() {
	defer func() {
		ws.closed = true
		close(ws.send)
		ws.shutdown()
	}()

	ws.conn.SetReadLimit(maxMessageSize)
	_ = ws.conn.SetReadDeadline(time.Now().Add(pongTime))
	if ws.pingPong {
		ws.conn.SetPongHandler(func(string) error { _ = ws.conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
	} else {
		ws.conn.SetPingHandler(func(string) error {
			_ = ws.conn.SetReadDeadline(time.Now().Add(pongTime))
			err := ws.conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(writeWait))
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
				ws.log.Error().Err(err).Msg("WebSocket read fail")
			}
			break
		}
		ws.OnMessage(message, err)
	}
}

// writer pumps messages from the send channel to the websocket connection.
// Blocking, must be called as goroutine. Serializes all websocket writes.
func (ws *WS) writer() {
	defer ws.shutdown()

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
		_ = ws.conn.write(websocket.CloseMessage, []byte{})
		return false
	}
	if err := ws.conn.write(websocket.TextMessage, message); err != nil {
		return false
	}
	return true
}

func newSocket(conn *websocket.Conn, pingPong bool, log *logger.Logger) *WS {
	return &WS{
		conn:      deadlineConn{Conn: conn, wt: writeWait},
		send:      make(chan []byte),
		once:      sync.Once{},
		Done:      make(chan struct{}, 1),
		pingPong:  pingPong,
		OnMessage: func(message []byte, err error) {},
		log:       log,
	}
}

func (ws *WS) Listen() {
	go ws.writer()
	go ws.reader()
}

func (ws *WS) Write(data []byte) {
	if !ws.closed {
		ws.send <- data
	}
}

func (ws *WS) Close() { _ = ws.conn.write(websocket.CloseMessage, []byte{}) }

func (ws *WS) shutdown() {
	ws.once.Do(func() {
		_ = ws.conn.Close()
		close(ws.Done)
		ws.log.Debug().Msg("WebSocket should be closed now")
	})
}
