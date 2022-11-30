package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/comm"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

type Coordinator struct {
	comm.SocketClient
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
	conn, err := comm.NewConnector().NewClient(address, log)
	if err != nil {
		return nil, err
	}
	return &Coordinator{SocketClient: comm.New(conn, "c", id, log)}, nil
}

func (c *Coordinator) HandleRequests(h *Handler) {
	ap, err := webrtc.NewApiFactory(h.conf.Webrtc, c.Log, nil)
	if err != nil {
		c.Log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}

	c.OnPacket(func(p comm.In) {
		var err error
		switch p.T {
		case api.TerminateSession:
			dat := api.Unwrap[api.TerminateSessionRequest](p.Payload)
			if dat == nil {
				err = api.ErrMalformed
				break
			}
			c.Log.Info().Msgf("Received a terminate session [%v]", dat.Id)
			c.HandleTerminateSession(*dat, h)
		case api.WebrtcInit:
			c.Log.Info().Msg("Received a request to createOffer from browser via coordinator")
			var out comm.Out
			if dat := api.Unwrap[api.WebrtcInitRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				out = c.HandleWebrtcInit(*dat, h, ap)
			}
			h.cord.Route(p, out)
		case api.WebrtcAnswer:
			c.Log.Info().Msg("Received answer SDP from browser")
			dat := api.Unwrap[api.WebrtcAnswerRequest](p.Payload)
			if dat == nil {
				err = api.ErrMalformed
				break
			}
			c.HandleWebrtcAnswer(*dat, h)
		case api.WebrtcIceCandidate:
			c.Log.Info().Msg("Received remote Ice Candidate from browser")
			dat := api.Unwrap[api.WebrtcIceCandidateRequest](p.Payload)
			if dat == nil {
				err = api.ErrMalformed
				break
			}
			c.HandleWebrtcIceCandidate(*dat, h)
		case api.StartGame:
			c.Log.Info().Msg("Received game start request")
			var out comm.Out
			if dat := api.Unwrap[api.StartGameRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				out = c.HandleGameStart(*dat, h)
			}
			h.cord.Route(p, out)
		case api.QuitGame:
			c.Log.Info().Msg("Received game quit request")
			dat := api.Unwrap[api.GameQuitRequest](p.Payload)
			if dat == nil {
				err = api.ErrMalformed
				break
			}
			c.HandleQuitGame(*dat, h)
		case api.SaveGame:
			c.Log.Info().Msg("Received a save game from coordinator")
			var out comm.Out
			if dat := api.Unwrap[api.SaveGameRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				out = c.HandleSaveGame(*dat, h)
			}
			h.cord.Route(p, out)
		case api.LoadGame:
			c.Log.Info().Msg("Received load game request")
			var out comm.Out
			if dat := api.Unwrap[api.LoadGameRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				out = c.HandleLoadGame(*dat, h)
			}
			h.cord.Route(p, out)
		case api.ChangePlayer:
			c.Log.Info().Msg("Received an update player index request")
			var out comm.Out
			if dat := api.Unwrap[api.ChangePlayerRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				out = c.HandleChangePlayer(*dat, h)
			}
			h.cord.Route(p, out)
		case api.ToggleMultitap:
			c.Log.Info().Msg("Received multitap toggle request")
			var out comm.Out
			if dat := api.Unwrap[api.ToggleMultitapRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				c.HandleToggleMultitap(*dat, h)
			}
			h.cord.Route(p, out)
		case api.RecordGame:
			c.Log.Info().Msg("Received recording request")
			var out comm.Out
			if dat := api.Unwrap[api.RecordGameRequest](p.Payload); dat == nil {
				err, out = api.ErrMalformed, comm.EmptyPacket
			} else {
				c.HandleRecordGame(*dat, h)
			}
			h.cord.Route(p, out)
		default:
			c.Log.Warn().Msgf("unhandled packet type %v", p.T)
		}
		if err != nil {
			c.Log.Error().Err(err).Msgf("malformed packet #%v", p.T)
		}
	})
}

func (c *Coordinator) CloseRoom(id string) { c.Notify(api.CloseRoom, id) }

func (c *Coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }

func (c *Coordinator) IceCandidate(candidate string, sessionId network.Uid) {
	c.Notify(api.NewWebrtcIceCandidateRequest(sessionId, candidate))
}
