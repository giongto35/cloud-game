package coordinator

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client

	WorkerID network.Uid
	Address  string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer     string
	StunTurnServer string
	IsAvailable    bool
	Zone           string
}

// NewWorkerClient returns a client connecting to worker.
// This connection exchanges information between workers and server.
func NewWorkerClient(c *websocket.Conn, workerID network.Uid) *WorkerClient {
	return &WorkerClient{
		Client:      cws.NewClient(c),
		WorkerID:    workerID,
		IsAvailable: true,
	}
}

func (wc *WorkerClient) makeAvailable(avail bool) { wc.IsAvailable = avail }

func (wc *WorkerClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Worker [%s] %s", wc.WorkerID.Short(), format), args...)
}

func (wc *WorkerClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Worker [%s] %s", wc.WorkerID.Short(), fmt.Sprint(args...)))
}
