package api

import "github.com/giongto35/cloud-game/v2/pkg/cws"

const (
	ServerId      = "server_id"
	SignalReady   = "signal_ready"
	ConfPushRoute = "conf_push"
)

type ConfPushCall struct {
	Data []byte `json:"data"`
}

func (packet *ConfPushCall) From(data string) error { return from(packet, data) }
func (packet *ConfPushCall) To() (string, error)    { return to(packet) }

func ServerIdPacket(id string) cws.WSPacket        { return cws.WSPacket{ID: ServerId, Data: id} }
func ConfigRequestPacket(conf string) cws.WSPacket { return cws.WSPacket{Data: conf} }
