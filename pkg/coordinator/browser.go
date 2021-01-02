package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

type BrowserClient struct {
	*cws.Client
	SessionID string
	RoomID    string
	WorkerID  string // TODO: how about pointer to workerClient?
}

// NewCoordinatorClient returns a client connecting to browser.
// This connection exchanges information between browser and coordinator.
func NewBrowserClient(c *websocket.Conn, browserID string) *BrowserClient {
	return &BrowserClient{
		Client:    cws.NewClient(c),
		SessionID: browserID,
	}
}

// Register new log
func (bc *BrowserClient) Printf(format string, args ...interface{}) {
	newFmt := fmt.Sprintf("Browser %s] %s", bc.SessionID, format)
	log.Printf(newFmt, args...)
}

func (bc *BrowserClient) Println(args ...interface{}) {
	msg := fmt.Sprintf("Browser %s] %s", bc.SessionID, fmt.Sprint(args...))
	log.Println(msg)
}
