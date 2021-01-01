package worker

import "github.com/giongto35/cloud-game/v2/pkg/cws/api"

func (h *Handler) routes() {
	if h.oClient == nil {
		return
	}

	h.oClient.Receive(api.ServerId, h.handleServerId())
	h.oClient.Receive(api.SignalReady, h.handleSignalReady())
	h.oClient.Receive(api.ConfPushRoute, h.handleConfPush())
}
