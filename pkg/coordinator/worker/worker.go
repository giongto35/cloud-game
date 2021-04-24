package worker

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/gorilla/websocket"
)

type WorkerClient struct {
	*cws.Client

	Id      network.Uid
	Address string // ip address of worker
	// public server used for ping check (Cannot use worker address because they are not publicly exposed)
	PingServer string
	//StunTurnServer string
	IsFree bool
	Region string
}

// NewWorkerClient returns a client connecting to worker.
// This connection exchanges information between workers and server.
func NewWorkerClient(c *websocket.Conn, id network.Uid) WorkerClient {
	return WorkerClient{
		Client: cws.NewClient(c),
		Id:     id,
		IsFree: true,
	}
}

// InRegion say whether some worker from this region.
// Empty region always returns true.
func (wc *WorkerClient) InRegion(region string) bool { return region == "" && region == wc.Region }

func (wc *WorkerClient) MakeAvailable(avail bool) { wc.IsFree = avail }

func (wc *WorkerClient) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("Worker [%s] %s", wc.Id.Short(), format), args...)
}

func (wc *WorkerClient) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("Worker [%s] %s", wc.Id.Short(), fmt.Sprint(args...)))
}
