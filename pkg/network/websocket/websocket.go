package websocket

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// be aware of ReadBufferSize, WriteBufferSize (default 4096)
// https://pkg.go.dev/github.com/gorilla/websocket?tab=doc#Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
}

func Connect(address url.URL) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	ws, _, err := dialer.Dial(address.String(), nil)
	return ws, err
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}
