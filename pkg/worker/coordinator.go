package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type Coordinator struct {
	client.DefaultClient
}

func NewCoordinator(conn *ipc.Client) Coordinator {
	return Coordinator{DefaultClient: client.New(conn, "cord")}
}

func (c *Coordinator) HandleRequests(h *Handler) {
	c.OnPacket(func(p ipc.InPacket) {
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
			c.Printf("warning: unhandled packet type %v", p.T)
		}
	})
}

func (c *Coordinator) CloseRoom(id string) {
	// api.CloseRoom
	_ = c.SendAndForget(api.CloseRoom, id)
}

func (c *Coordinator) RegisterRoom(id string) {
	// api.RegisterRoom
	_ = c.SendAndForget(api.RegisterRoom, id)
}

func (c *Coordinator) IceCandidate(candidate string, sessionId string) {
	//api.IceCandidate
	_ = c.SendAndForget(api.IceCandidate, api.WebrtcIceCandidateRequest{
		StatefulRequest: api.StatefulRequest{Id: network.Uid(sessionId)},
		Candidate:       candidate,
	})
}
