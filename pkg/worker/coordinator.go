package worker

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type Coordinator struct {
	id   network.Uid
	wire *ipc.Client
}

func NewCoordinator(conn *ipc.Client) Coordinator {
	return Coordinator{id: network.NewUid(), wire: conn}
}

func (c *Coordinator) HandleRequests(h *Handler) {
	c.wire.OnPacket = func(p ipc.InPacket) {
		switch p.T {
		case api.IdentifyWorker:
			c.HandleIdentifyWorker(p.Payload, h)
		case api.TerminateSession:
			c.HandleTerminateSession(p.Payload, h)
		case api.WebrtcInit:
			c.HandleWebrtcInit(p, h)
		case api.WebrtcAnswer:
			c.HandleWebrtcAnswer(p, h)
		case api.WebrtcIceCandidate:
			c.HandleWebrtcIceCandidate(p, h)
		case api.StartGame:
			c.HandleGameStart(p, h)
		case api.QuitGame:
			c.HandleQuitGame(p, h)
		case api.SaveGame:
			c.HandleSaveGame(p, h)
		case api.LoadGame:
			c.HandleLoadGame(p, h)
		case api.ChangePlayer:
			c.HandleChangePlayer(p, h)
		case api.ToggleMultitap:
			c.HandleToggleMultitap(p, h)
		default:
			c.Println("warning: unhandled packet type %v", p.T)
		}
	}
}

func (c *Coordinator) Printf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("cord: [%s] %s", c.id.Short(), format), args...)
}

func (c *Coordinator) Println(args ...interface{}) {
	log.Println(fmt.Sprintf("cord: [%s] %s", c.id.Short(), fmt.Sprint(args...)))
}

func (c *Coordinator) WaitDisconnect() {
	<-c.wire.Conn.Done
}

func (c *Coordinator) CloseRoom(id string) {
	// api.CloseRoom
	_ = c.wire.Send(api.CloseRoom, api.CloseRoomRequest(id))
}

func (c *Coordinator) RegisterRoom(id string) {
	// api.RegisterRoom
	_ = c.wire.Send(api.RegisterRoom, api.RegisterRoomRequest(id))
}

func (c *Coordinator) IceCandidate(candidate string, sessionId string) {
	//api.IceCandidate
	_ = c.wire.Send(api.IceCandidate, api.WebrtcIceCandidateRequest{
		StatefulRequest: api.StatefulRequest{Id: network.Uid(sessionId)},
		Candidate:       candidate,
	})
}
