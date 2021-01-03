package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client

	WorkerID string
	Address  string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer     string
	StunTurnServer string
	IsAvailable    bool
	Zone           string
}

// NewWorkerClient returns a client connecting to worker.
// This connection exchanges information between workers and server.
func NewWorkerClient(c *websocket.Conn, workerID string) *WorkerClient {
	return &WorkerClient{
		Client:      cws.NewClient(c),
		WorkerID:    workerID,
		IsAvailable: true,
	}
}

func (wc *WorkerClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Worker %s] %s", wc.WorkerID, format), args...)
}

func (wc *WorkerClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Worker %s] %s", wc.WorkerID, fmt.Sprint(args...)))
}
