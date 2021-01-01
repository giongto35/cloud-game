package worker

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
)

func (h *Handler) handleServerId() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("[worker] <- got id: %s", resp.Data)
		h.serverID = resp.Data
		h.w.lock.Unlock()
		return
	}
}

func (h *Handler) handleSignalReady() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("[cws] <- signal 'ready'")
		return
	}
}

func (h *Handler) handleConfPush() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("[cws] <- config push")

		call := api.ConfPushCall{}
		if err := call.From(resp.Data); err != nil {
			return cws.EmptyPacket
		}

		// parse config

		// non-blocking reload
		go h.Prepare()

		return req
	}
}
