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
		case api.TerminateSession:
			c.HandleTerminateSession(p.Payload, h)
		case api.WebrtcInit:
			c.Printf("Received a request to createOffer from browser via coordinator")
			c.HandleWebrtcInit(p, h)
		case api.WebrtcAnswer:
			c.Printf("Received answer SDP from browser")
			c.HandleWebrtcAnswer(p, h)
		case api.WebrtcIceCandidate:
			c.Printf("Received remote Ice Candidate from browser")
			c.HandleWebrtcIceCandidate(p, h)
		case api.StartGame:
			c.Printf("Received game start request")
			c.HandleGameStart(p, h)
		case api.QuitGame:
			c.Printf("Received game quit request")
			c.HandleQuitGame(p, h)
		case api.SaveGame:
			c.Printf("Received a save game from coordinator")
			c.HandleSaveGame(p, h)
		case api.LoadGame:
			c.Printf("Received load game request")
			c.HandleLoadGame(p, h)
		case api.ChangePlayer:
			c.Printf("Received an update player index request")
			c.HandleChangePlayer(p, h)
		case api.ToggleMultitap:
			c.Printf("Received multitap toggle request")
			c.HandleToggleMultitap(p, h)
		default:
			c.Printf("warning: unhandled packet type %v", p.T)
		}
	})
}

func (c *Coordinator) CloseRoom(id string) { _ = c.SendAndForget(api.CloseRoom, id) }

func (c *Coordinator) RegisterRoom(id string) { _ = c.SendAndForget(api.RegisterRoom, id) }

func (c *Coordinator) IceCandidate(candidate string, sessionId string) {
	_ = c.SendAndForget(api.IceCandidate, api.WebrtcIceCandidateRequest{
		StatefulRequest: api.StatefulRequest{Id: network.Uid(sessionId)},
		Candidate:       candidate,
	})
}
