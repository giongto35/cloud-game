package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

// workerRoutes adds all worker request routes.
func (c *Server) workerRoutes(wc *WorkerClient) {
	if wc == nil {
		return
	}
	wc.Receive(api.ConfigRequest, wc.handleConfigRequest())
	wc.Receive(api.Heartbeat, wc.handleHeartbeat())
	wc.Receive(api.RegisterRoom, wc.handleRegisterRoom(c))
	wc.Receive(api.GetRoom, wc.handleGetRoom(c))
	wc.Receive(api.CloseRoom, wc.handleCloseRoom(c))
	wc.Receive(api.IceCandidate, wc.handleIceCandidate(c))
}

// useragentRoutes adds all useragent (browser) request routes.
func (c *Server) useragentRoutes(bc *BrowserClient) {
	if bc == nil {
		return
	}
	bc.Receive(api.Heartbeat, bc.handleHeartbeat())
	bc.Receive(api.InitWebrtc, bc.handleInitWebrtc(c))
	bc.Receive(api.Answer, bc.handleAnswer(c))
	bc.Receive(api.IceCandidate, bc.handleIceCandidate(c))
	bc.Receive(api.GameStart, bc.handleGameStart(c))
	bc.Receive(api.GameQuit, bc.handleGameQuit(c))
	bc.Receive(api.GameSave, bc.handleGameSave(c))
	bc.Receive(api.GameLoad, bc.handleGameLoad(c))
	bc.Receive(api.GamePlayerSelect, bc.handleGamePlayerSelect(c))
	bc.Receive(api.GameMultitap, bc.handleGameMultitap(c))
}
