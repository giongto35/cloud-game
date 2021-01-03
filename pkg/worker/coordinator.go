package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/gorilla/websocket"
)

// CoordinatorClient maintains connection to coordinator.
// We expect only one CoordinatorClient for each server.
type CoordinatorClient struct {
	*cws.Client
}

// NewCoordinatorClient returns a client connecting to coordinator
// for coordination between different server.
func NewCoordinatorClient(oc *websocket.Conn) *CoordinatorClient {
	if oc == nil {
		return nil
	}

	oClient := &CoordinatorClient{
		Client: cws.NewClient(oc),
	}
	return oClient
}
