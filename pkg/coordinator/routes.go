package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

// routes adds all worker request routes
func (o *Server) routes(wc *WorkerClient) {
	if o == nil {
		return
	}

	wc.Receive(api.ConfigRequest, wc.handleConfigRequest())
	wc.Receive("heartbeat", wc.handleHeartbeat())
	wc.Receive("registerRoom", wc.handleRegisterRoom(o))
	wc.Receive(api.GetRoom, wc.handleGetRoom(o))
	wc.Receive(api.CloseRoom, wc.handleCloseRoom(o))
	wc.Receive("candidate", wc.handleIceCandidate(o))
}
