package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
)

type coordinator struct {
	com.SocketClient
}

var connector = com.NewConnector()

// connect to a coordinator.
func connect(host string, conf worker.Worker, addr string, log *logger.Logger) (*coordinator, error) {
	scheme := "ws"
	if conf.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Network.Endpoint}

	log.Info().Str("c", "c").Str("d", "→").Msgf("Handshake %s", address.String())

	id := network.NewUid()
	req, err := buildConnQuery(id, conf, addr)
	if req != "" && err == nil {
		address.RawQuery = "data=" + req
	}
	conn, err := connector.NewClient(address, log)
	if err != nil {
		return nil, err
	}
	return &coordinator{com.New(conn, "c", id, log)}, nil
}

func (c *coordinator) HandleRequests(s *Service) {
	ap, err := webrtc.NewApiFactory(s.conf.Webrtc, c.Log, nil)
	if err != nil {
		c.Log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}
	c.ProcessMessages()
	skipped := com.Out{}

	c.OnPacket(func(x com.In) (err error) {
		var out com.Out
		switch x.T {
		case api.WebrtcInit:
			if dat := api.Unwrap[api.WebrtcInitRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleWebrtcInit(*dat, s, ap)
			}
		case api.WebrtcAnswer:
			dat := api.Unwrap[api.WebrtcAnswerRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcAnswer(*dat, s)
		case api.WebrtcIce:
			dat := api.Unwrap[api.WebrtcIceCandidateRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcIceCandidate(*dat, s)
		case api.StartGame:
			if dat := api.Unwrap[api.StartGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleGameStart(*dat, s)
			}
		case api.TerminateSession:
			dat := api.Unwrap[api.TerminateSessionRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleTerminateSession(*dat, s)
		case api.QuitGame:
			dat := api.Unwrap[api.GameQuitRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleQuitGame(*dat, s)
		case api.SaveGame:
			if dat := api.Unwrap[api.SaveGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleSaveGame(*dat, s)
			}
		case api.LoadGame:
			if dat := api.Unwrap[api.LoadGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleLoadGame(*dat, s)
			}
		case api.ChangePlayer:
			if dat := api.Unwrap[api.ChangePlayerRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleChangePlayer(*dat, s)
			}
		case api.ToggleMultitap:
			if dat := api.Unwrap[api.ToggleMultitapRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleToggleMultitap(*dat, s)
			}
		case api.RecordGame:
			if dat := api.Unwrap[api.RecordGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleRecordGame(*dat, s)
			}
		default:
			c.Log.Warn().Msgf("unhandled packet type %v", x.T)
		}
		if out != skipped {
			s.cord.Route(x, out)
		}
		return nil
	})
}

func (c *coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }

// CloseRoom sends a signal to coordinator which will remove that room from its list.
func (c *coordinator) CloseRoom(id string) { c.Notify(api.CloseRoom, id) }
func (c *coordinator) IceCandidate(candidate string, sessionId network.Uid) {
	c.Notify(api.NewWebrtcIceCandidateRequest(sessionId, candidate))
}
