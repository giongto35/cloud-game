package worker

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

func (h *Handler) routes() {
	if h.oClient == nil {
		return
	}

	h.oClient.Receive(api.ServerId, h.handleServerId())
	h.oClient.Receive(api.TerminateSession, h.handleTerminateSession())
	h.oClient.Receive(api.InitWebrtc, h.handleInitWebrtc())
	h.oClient.Receive(api.Answer, h.handleAnswer())
	h.oClient.Receive(api.IceCandidate, h.handleIceCandidate())

	h.oClient.Receive(api.GameStart, h.handleGameStart())
	h.oClient.Receive(api.GameQuit, h.handleGameQuit())
	h.oClient.Receive(api.GameSave, h.handleGameSave())
	h.oClient.Receive(api.GameLoad, h.handleGameLoad())
	h.oClient.Receive(api.GamePlayerSelect, h.handleGamePlayerSelect())
	h.oClient.Receive(api.GameMultitap, h.handleGameMultitap())
	h.oClient.Receive(api.GameRecording, h.handleGameRecording())
}
