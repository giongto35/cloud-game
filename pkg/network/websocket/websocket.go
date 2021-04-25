package websocket

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 10 * 1024
	pingTime       = pongTime * 9 / 10
	pongTime       = 60 * time.Second
	readWait       = 5 * time.Second
	writeWait      = 10 * time.Second
)

type WS struct {
	id   network.Uid
	conn deadlinedConn
	send chan []byte

	OnMessage WSMessageHandler

	pingPong bool

	shutdown *sync.WaitGroup
	Done     chan struct{}
}

type WSMessageHandler func(message []byte, err error)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	WriteBufferPool: &sync.Pool{},
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// reader pumps messages from the websocket connection to the OnMessage callback.
// Blocking, must be called as goroutine. Serializes all websocket reads.
func (ws *WS) reader() {
	defer func() {
		close(ws.send)
		ws.shutdown.Done()
		ws.close()
		log.Printf("%v [ws] CLOSE READER", ws.id.Short())
	}()
	ws.conn.setup(func(conn *websocket.Conn) {
		conn.SetReadLimit(maxMessageSize)
		if ws.pingPong {
			_ = conn.SetReadDeadline(time.Now().Add(pongTime))
			conn.SetPongHandler(func(string) error { _ = conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
		}
	})
	for {
		message, err := ws.conn.read()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			log.Printf("read error: %v", err)
			break
		}
		log.Printf("%v [ws] READ: %v", ws.id.Short(), string(message))
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
		log.Printf("%v [ws] CLOSE WRITER", ws.id.Short())
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
				} else {
					log.Printf("write error: %v", err)
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
	log.Printf("%v [ws] WRITE: %v", ws.id.Short(), string(message))
	if err := ws.conn.write(websocket.TextMessage, message); err != nil {
		return false
	}
	return true
}

// NewServer initializes new websocket peer requests handler.
func NewServer(w http.ResponseWriter, r *http.Request) *WS {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil
	}
	return newSocket(conn, true)
}

func NewClient(address url.URL) *WS {
	conn, _, err := websocket.DefaultDialer.Dial(address.String(), nil)
	if err != nil {
		return nil
	}
	return newSocket(conn, false)
}

func newSocket(conn *websocket.Conn, pingPong bool) *WS {
	// graceful shutdown ( ಠ_ಠ )
	shut := sync.WaitGroup{}
	shut.Add(2)

	safeConn := deadlinedConn{
		sock: conn,
		wt:   writeWait,
	}
	if !pingPong {
		safeConn.rt = readWait
	}

	ws := &WS{
		id:       network.NewUid(),
		conn:     safeConn,
		send:     make(chan []byte),
		shutdown: &shut,
		Done:     make(chan struct{}, 1),
	}

	go ws.writer()
	go ws.reader()

	return ws
}

func (ws *WS) Write(data []byte) { ws.send <- data }

func (ws *WS) Close() {
	log.Printf("%v [ws] SEND CLOSE MESSAGE", ws.id.Short())
	_ = ws.conn.write(websocket.CloseMessage, []byte{})
	log.Printf("%v [ws] CLOSE", ws.id.Short())
}

func (ws *WS) close() {
	ws.shutdown.Wait()
	_ = ws.conn.close()
	ws.Done <- struct{}{}
}
