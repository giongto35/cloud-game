package websocket

import (
	"crypto/tls"
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

type WS struct {
	conn deadlinedConn
	send chan []byte

	OnMessage WSMessageHandler

	pingPong bool

	shutdown *sync.WaitGroup
	once     sync.Once
	Done     chan struct{}
	closed   bool
	log      *logger.Logger
}

type WSMessageHandler func(message []byte, err error)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	WriteBufferPool: &sync.Pool{},
}

// reader pumps messages from the websocket connection to the OnMessage callback.
// Blocking, must be called as goroutine. Serializes all websocket reads.
func (ws *WS) reader() {
	defer func() {
		ws.closed = true
		close(ws.send)
		ws.shutdown.Done()
		ws.close()
	}()

	ws.conn.setup(func(conn *websocket.Conn) {
		conn.SetReadLimit(maxMessageSize)
		_ = conn.SetReadDeadline(time.Now().Add(pongTime))
		if ws.pingPong {
			conn.SetPongHandler(func(string) error { _ = conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
		} else {
			conn.SetPingHandler(func(string) error {
				_ = conn.SetReadDeadline(time.Now().Add(pongTime))
				err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(writeWait))
				if err == websocket.ErrCloseSent {
					return nil
				} else if e, ok := err.(net.Error); ok && e.Temporary() {
					return nil
				}
				return err
			})
		}
	})
	for {
		message, err := ws.conn.read()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.log.Error().Err(err).Msg("unexpected error")
			} else {
				ws.log.Error().Err(err).Msg("read error")
			}
			break
		}
		ws.OnMessage(message, err)
	}
}

// writer pumps messages from the send channel to the websocket connection.
// Blocking, must be called as goroutine. Serializes all websocket writes.
func (ws *WS) writer() {
	var ticker *time.Ticker
	if ws.pingPong {
		ticker = time.NewTicker(pingTime)
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
		ws.shutdown.Done()
		ws.close()
	}()
	if ws.pingPong {
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
		for {
			select {
			case message, ok := <-ws.send:
				if !ws.handleMessage(message, ok) {
					return
				}
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

// NewServer initializes new websocket peer requests handler.
func NewServer(w http.ResponseWriter, r *http.Request, log *logger.Logger) (*WS, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
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

func newSocket(conn *websocket.Conn, pingPong bool, log *logger.Logger) *WS {
	// graceful shutdown ( ಠ_ಠ )
	shut := sync.WaitGroup{}
	shut.Add(2)

	safeConn := deadlinedConn{sock: conn, wt: writeWait}

	ws := &WS{
		conn:      safeConn,
		send:      make(chan []byte),
		shutdown:  &shut,
		once:      sync.Once{},
		Done:      make(chan struct{}, 1),
		pingPong:  pingPong,
		OnMessage: func(message []byte, err error) {},
		log:       log,
	}

	go ws.writer()
	go ws.reader()

	return ws
}

func (ws *WS) Write(data []byte) {
	if !ws.closed {
		ws.send <- data
	}
}

func (ws *WS) Close() { _ = ws.conn.write(websocket.CloseMessage, []byte{}) }

func (ws *WS) close() {
	ws.shutdown.Wait()
	ws.once.Do(func() {
		_ = ws.conn.close()
		ws.Done <- struct{}{}
		ws.log.Debug().Msg("WebSocket should be closed now")
	})
}

func (ws *WS) GetRemoteAddr() net.Addr { return ws.conn.sock.RemoteAddr() }
