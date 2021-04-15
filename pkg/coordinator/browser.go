package coordinator

import (
	"fmt"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client

	RoomID    string
	SessionID network.Uid
	WorkerID  network.Uid // TODO: how about pointer to workerClient?
}

// NewCoordinatorClient returns a client connecting to browser.
// This connection exchanges information between browser and coordinator.
func NewBrowserClient(c *websocket.Conn, browserID network.Uid) *BrowserClient {
	return &BrowserClient{
		Client:    cws.NewClient(c),
		SessionID: browserID,
	}
}

// Register new log
func (bc *BrowserClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Browser [%s] %s", bc.SessionID.Short(), format), args...)
}

func (bc *BrowserClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Browser [%s] %s", bc.SessionID.Short(), fmt.Sprint(args...)))
}
