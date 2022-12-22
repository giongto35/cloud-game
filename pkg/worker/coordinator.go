package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

type Coordinator struct {
	client.SocketClient

	log *logger.Logger
}

func newCoordinatorConnection(host string, conf worker.Worker, addr string, log *logger.Logger) (Coordinator, error) {
	scheme := "ws"
	if conf.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Network.Endpoint}

	id := network.NewUid()
	req, err := MakeConnectionRequest(id.String(), conf, addr)
	if req != "" && err == nil {
		address.RawQuery = "data=" + req
	}

	conn, err := ipc.NewClient(address, log)
	if err != nil {
		return Coordinator{}, err
	}
	return Coordinator{SocketClient: client.NewWithId(id, conn, "c", log), log: log}, nil
}

func (c *Coordinator) HandleRequests(h *Handler) {
	ap, err := webrtc.NewApiFactory(h.conf.Webrtc, c.log, nil)
	if err != nil {
		c.log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}

	c.OnPacket(func(p ipc.InPacket) {
		switch p.T {
		case api.TerminateSession:
			c.HandleTerminateSession(p.Payload, h)
		case api.WebrtcInit:
			c.log.Info().Msg("Received a request to createOffer from browser via coordinator")
			c.HandleWebrtcInit(p, h, ap)
		case api.WebrtcAnswer:
			c.log.Info().Msg("Received answer SDP from browser")
			c.HandleWebrtcAnswer(p, h)
		case api.WebrtcIceCandidate:
			c.log.Info().Msg("Received remote Ice Candidate from browser")
			c.HandleWebrtcIceCandidate(p, h)
		case api.StartGame:
			c.log.Info().Msg("Received game start request")
			c.HandleGameStart(p, h)
		case api.QuitGame:
			c.log.Info().Msg("Received game quit request")
			c.HandleQuitGame(p, h)
		case api.SaveGame:
			c.log.Info().Msg("Received a save game from coordinator")
			c.HandleSaveGame(p, h)
		case api.LoadGame:
			c.log.Info().Msg("Received load game request")
			c.HandleLoadGame(p, h)
		case api.ChangePlayer:
			c.log.Info().Msg("Received an update player index request")
			c.HandleChangePlayer(p, h)
		case api.ToggleMultitap:
			c.log.Info().Msg("Received multitap toggle request")
			c.HandleToggleMultitap(p, h)
		case api.RecordGame:
			c.log.Info().Msg("Received recording request")
			c.HandleRecordGame(p, h)
		default:
			c.log.Warn().Msgf("unhandled packet type %v", p.T)
		}
	})
}

func (c *Coordinator) CloseRoom(id string) { _ = c.SendAndForget(api.CloseRoom, id) }

func (c *Coordinator) RegisterRoom(id string) { _ = c.SendAndForget(api.RegisterRoom, id) }

func (c *Coordinator) IceCandidate(candidate string, sessionId string) {
	_ = c.SendAndForget(api.IceCandidate, api.WebrtcIceCandidateRequest{
		Stateful:  api.Stateful{Id: network.Uid(sessionId)},
		Candidate: candidate,
	})
}
