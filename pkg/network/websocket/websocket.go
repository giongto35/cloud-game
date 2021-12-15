package websocket

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type Upgrader struct {
	websocket.Upgrader

	origin string
}

var DefaultUpgrader = Upgrader{
	Upgrader: websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: false,
	},
}

func NewUpgrader(origin string) Upgrader {
	u := DefaultUpgrader
	switch {
	case origin == "*":
		u.CheckOrigin = func(r *http.Request) bool { return true }
	case origin != "":
		u.CheckOrigin = func(r *http.Request) bool { return r.Header.Get("Origin") == origin }
	}
	return u
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	if u.origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", u.origin)
	}
	return u.Upgrader.Upgrade(w, r, responseHeader)
}

func Connect(address url.URL) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	ws, _, err := dialer.Dial(address.String(), nil)
	return ws, err
}
