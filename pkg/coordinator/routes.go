package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

// workerRoutes adds all worker request routes.
func (s *Server) workerRoutes(wc *WorkerClient) {
	if wc == nil {
		return
	}
	wc.Receive(api.Heartbeat, wc.handleHeartbeat())
	wc.Receive(api.RegisterRoom, wc.handleRegisterRoom(s))
	wc.Receive(api.GetRoom, wc.handleGetRoom(s))
	wc.Receive(api.CloseRoom, wc.handleCloseRoom(s))
	wc.Receive(api.IceCandidate, wc.handleIceCandidate(s))
}

// useragentRoutes adds all useragent (browser) request routes.
func (s *Server) useragentRoutes(bc *BrowserClient) {
	if bc == nil {
		return
	}
	bc.Receive(api.Heartbeat, bc.handleHeartbeat())
	bc.Receive(api.InitWebrtc, bc.handleInitWebrtc(s))
	bc.Receive(api.Answer, bc.handleAnswer(s))
	bc.Receive(api.IceCandidate, bc.handleIceCandidate(s))
	bc.Receive(api.GameStart, bc.handleGameStart(s))
	bc.Receive(api.GameQuit, bc.handleGameQuit(s))
	bc.Receive(api.GameSave, bc.handleGameSave(s))
	bc.Receive(api.GameLoad, bc.handleGameLoad(s))
	bc.Receive(api.GamePlayerSelect, bc.handleGamePlayerSelect(s))
	bc.Receive(api.GameMultitap, bc.handleGameMultitap(s))
	bc.Receive(api.GameRecording, bc.handleGameRecording(s))
}
