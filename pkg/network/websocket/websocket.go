package websocket

import (
	"crypto/tls"
	"net/url"

	"github.com/gorilla/websocket"
)

func Connect(address url.URL) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	if address.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	ws, _, err := dialer.Dial(address.String(), nil)
	return ws, err
}
