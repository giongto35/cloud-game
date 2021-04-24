package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

type deadlinedConn struct {
	sock *websocket.Conn
	rt   time.Duration
	wt   time.Duration
}

func (conn *deadlinedConn) setup(fn func(conn *websocket.Conn)) { fn(conn.sock) }

func (conn *deadlinedConn) close() error { return conn.sock.Close() }

func (conn *deadlinedConn) read() (message []byte, err error) {
	if conn.rt > 0 {
		if er := conn.sock.SetReadDeadline(time.Now().Add(conn.rt)); er != nil {
			return []byte{}, er
		}
	}
	_, message, err = conn.sock.ReadMessage()
	return
}

func (conn *deadlinedConn) write(t int, mess []byte) error {
	if err := conn.sock.SetWriteDeadline(time.Now().Add(conn.wt)); err != nil {
		return err
	}
	return conn.sock.WriteMessage(t, mess)
}
