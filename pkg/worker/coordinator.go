package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

type Coordinator struct {
	client.SocketClient

	log *logger.Logger
}

func newCoordinatorConnection(host string, conf worker.Worker, addr string, log *logger.Logger) (*Coordinator, error) {
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
	conn, err := client.NewConnector().NewClient(address, log)
	if err != nil {
		return nil, err
	}
	return &Coordinator{SocketClient: client.NewWithId(id, conn, "c", log), log: log}, nil
}

func (c *Coordinator) HandleRequests(h *Handler) {
	ap, err := webrtc.NewApiFactory(h.conf.Webrtc, c.log, nil)
	if err != nil {
		c.log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}

	c.OnPacket(func(p client.InPacket) {
		switch p.T {
		case api.TerminateSession:
			resp, err := api.Unwrap[api.TerminateSessionRequest](p.Payload)
			if err != nil {
				c.log.Error().Err(err).Msg("terminate session error")
				return
			}
			c.log.Info().Msgf("Received a terminate session [%v]", resp.Id)
			c.HandleTerminateSession(*resp, h)
		case api.WebrtcInit:
			c.log.Info().Msg("Received a request to createOffer from browser via coordinator")
			c.HandleWebrtcInit(p, h, ap)
		case api.WebrtcAnswer:
			c.log.Info().Msg("Received answer SDP from browser")
			rq, err := api.Unwrap[api.WebrtcAnswerRequest](p.Payload)
			if err != nil {
				c.log.Error().Err(err).Msg("malformed WebRTC answer")
				return
			}
			c.HandleWebrtcAnswer(*rq, h)
		case api.WebrtcIceCandidate:
			c.log.Info().Msg("Received remote Ice Candidate from browser")
			rs, err := api.Unwrap[api.WebrtcIceCandidateRequest](p.Payload)
			if err != nil {
				c.log.Error().Err(err).Send()
				return
			}
			c.HandleWebrtcIceCandidate(*rs, h)
		case api.StartGame:
			c.log.Info().Msg("Received game start request")
			c.HandleGameStart(p, h)
		case api.QuitGame:
			c.log.Info().Msg("Received game quit request")
			resp, err := api.Unwrap[api.GameQuitRequest](p.Payload)
			if err != nil {
				c.log.Error().Err(err).Msg("malformed game quit request")
				return
			}
			c.HandleQuitGame(*resp, h)
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

func (c *Coordinator) CloseRoom(id string) { c.Notify(api.CloseRoom, id) }

func (c *Coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }

func (c *Coordinator) IceCandidate(candidate string, sessionId network.Uid) {
	c.Notify(api.NewWebrtcIceCandidateRequest(sessionId, candidate))
}
