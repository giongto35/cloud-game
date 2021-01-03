package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

// workerRoutes adds all worker request routes.
func (o *Server) workerRoutes(wc *WorkerClient) {
	if wc == nil {
		return
	}
	wc.Receive(api.ConfigRequest, wc.handleConfigRequest())
	wc.Receive(api.Heartbeat, wc.handleHeartbeat())
	wc.Receive(api.RegisterRoom, wc.handleRegisterRoom(o))
	wc.Receive(api.GetRoom, wc.handleGetRoom(o))
	wc.Receive(api.CloseRoom, wc.handleCloseRoom(o))
	wc.Receive(api.IceCandidate, wc.handleIceCandidate(o))
}

// useragentRoutes adds all useragent (browser) request routes.
func (o *Server) useragentRoutes(bc *BrowserClient) {
	if bc == nil {
		return
	}
	bc.Receive(api.Heartbeat, bc.handleHeartbeat())
	bc.Receive(api.InitWebrtc, bc.handleInitWebrtc(o))
	bc.Receive(api.Answer, bc.handleAnswer(o))
	bc.Receive(api.IceCandidate, bc.handleIceCandidate(o))
	bc.Receive(api.GameStart, bc.handleGameStart(o))
	bc.Receive(api.GameQuit, bc.handleGameQuit(o))
	bc.Receive(api.GameSave, bc.handleGameSave(o))
	bc.Receive(api.GameLoad, bc.handleGameLoad(o))
	bc.Receive(api.GamePlayerSelect, bc.handleGamePlayerSelect(o))
	bc.Receive(api.GameMultitap, bc.handleGameMultitap(o))
}
