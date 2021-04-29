package websocket

import (
	"log"
	"net"
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
	// TODO chack block until pongtime on disconnect
	pongTime  = 5 * time.Second
	readWait  = 2 * pongTime
	writeWait = 10 * time.Second
)

type WS struct {
	id   network.Uid
	conn deadlinedConn
	send chan []byte

	OnMessage WSMessageHandler

	pingPong bool

	shutdown *sync.WaitGroup
	Done     chan struct{}
	closed   bool
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
		ws.close("reader")
		log.Printf("%v [ws] CLOSE READER", ws.id.Short())
	}()

	ws.conn.setup(func(conn *websocket.Conn) {
		conn.SetReadLimit(maxMessageSize)
		if ws.pingPong {
			_ = conn.SetReadDeadline(time.Now().Add(pongTime))
			conn.SetPongHandler(func(string) error { _ = conn.SetReadDeadline(time.Now().Add(pongTime)); return nil })
		} else {
			//_ = conn.SetReadDeadline(time.Now().Add(pongTime))
			//conn.SetPingHandler(func(m string) error {
			//	_ = conn.SetReadDeadline(time.Now().Add(pongTime))
			//	err := conn.WriteControl(websocket.PongMessage, []byte(m), time.Now().Add(writeWait))
			//	if err == websocket.ErrCloseSent {
			//		return nil
			//	} else if e, ok := err.(net.Error); ok && e.Temporary() {
			//		return nil
			//	}
			//	return err
			//})
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
		ws.close("writer")
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
					log.Printf("PING FAILED")
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
	//log.Printf("%v [ws] WRITE: %v", ws.id.Short(), string(message))
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
		//safeConn.rt = readWait
	}

	ws := &WS{
		id:        network.NewUid(),
		conn:      safeConn,
		send:      make(chan []byte),
		shutdown:  &shut,
		Done:      make(chan struct{}, 1),
		pingPong:  pingPong,
		OnMessage: func(message []byte, err error) {},
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

func (ws *WS) Close() {
	log.Printf("%v [ws] SEND CLOSE MESSAGE", ws.id.Short())
	_ = ws.conn.write(websocket.CloseMessage, []byte{})
	log.Printf("%v [ws] CLOSE", ws.id.Short())
}

func (ws *WS) close(what string) {
	log.Printf("[%v] start close %v", ws.id, what)
	ws.shutdown.Wait()
	_ = ws.conn.close()
	ws.Done <- struct{}{}
	log.Printf("[%v] end close %v", ws.id, what)
}

func (ws *WS) GetRemoteAddr() net.Addr {
	return ws.conn.sock.RemoteAddr()
}
