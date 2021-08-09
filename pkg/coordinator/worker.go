package coordinator

import (
	"fmt"
	"log"
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client

	WorkerID string
	Address  string // ip address of worker
	// public server used for ping check
	PingServer     string
	StunTurnServer string
	userCount      int // may be atomic
	Zone           string

	mu sync.Mutex
}

// NewWorkerClient returns a client connecting to worker.
// This connection exchanges information between workers and server.
func NewWorkerClient(c *websocket.Conn, workerID string) *WorkerClient {
	return &WorkerClient{
		Client:   cws.NewClient(c),
		WorkerID: workerID,
	}
}

// ChangeUserQuantityBy increases or decreases the total amount of
// users connected to the current worker.
// We count users to determine when the worker becomes new game ready.
func (wc *WorkerClient) ChangeUserQuantityBy(n int) {
	wc.mu.Lock()
	wc.userCount += n
	// just to be on a safe side
	if wc.userCount < 0 {
		wc.userCount = 0
	}
	wc.mu.Unlock()
}

// HasGameSlot tells whether the current worker has a
// free slot to start a new game.
// Workers support only one game at a time.
func (wc *WorkerClient) HasGameSlot() bool {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.userCount == 0
}

func (wc *WorkerClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Worker %s] %s", wc.WorkerID, format), args...)
}

func (wc *WorkerClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Worker %s] %s", wc.WorkerID, fmt.Sprint(args...)))
}
