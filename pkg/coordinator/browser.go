package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client

	RoomID    string
	SessionID network.Uid
	Worker    *WorkerClient
}

// NewCoordinatorClient returns a client connecting to browser.
// This connection exchanges information between browser and coordinator.
func NewBrowserClient(c *websocket.Conn, browserID network.Uid) *BrowserClient {
	return &BrowserClient{
		Client:    cws.NewClient(c),
		SessionID: browserID,
	}
}

func (bc *BrowserClient) AssignWorker(w *WorkerClient) {
	bc.Worker = w
	w.makeAvailable(false)
}

func (bc *BrowserClient) RetainWorker() {
	if bc.Worker != nil {
		bc.Worker.IsAvailable = true
	}
}

// Register new log
func (bc *BrowserClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Browser [%s] %s", bc.SessionID.Short(), format), args...)
}

func (bc *BrowserClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Browser [%s] %s", bc.SessionID.Short(), fmt.Sprint(args...)))
}
