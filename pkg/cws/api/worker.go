package api

import (
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

const (
	Start            = "start"
	ServerId         = "server_id"
	TerminateSession = "terminateSession"
)

func ServerIdPacket(id network.Uid) cws.WSPacket   { return cws.WSPacket{ID: ServerId, Data: string(id)} }
func TerminateSessionPacket(sessionId network.Uid) cws.WSPacket {
	return cws.WSPacket{ID: TerminateSession, SessionID: sessionId}
}
func StartPacket(room string) cws.WSPacket { return cws.WSPacket{ID: Start, RoomID: room} }
