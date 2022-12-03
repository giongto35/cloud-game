package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

type coordinator struct {
	com.SocketClient
}

// connect to a coordinator.
func connect(host string, conf worker.Worker, addr string, log *logger.Logger) (*coordinator, error) {
	scheme := "ws"
	if conf.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Network.Endpoint}

	log.Info().Str("c", "c").Str("d", "→").Msgf("Handshake %s", address.String())

	id := network.NewUid()
	req, err := MakeConnectionRequest(id.String(), conf, addr)
	if req != "" && err == nil {
		address.RawQuery = "data=" + req
	}
	conn, err := com.NewConnector().NewClient(address, log)
	if err != nil {
		return nil, err
	}
	return &coordinator{SocketClient: com.New(conn, "c", id, log)}, nil
}

func (c *coordinator) HandleRequests(h *Service) {
	ap, err := webrtc.NewApiFactory(h.conf.Webrtc, c.Log, nil)
	if err != nil {
		c.Log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}
	c.OnPacket(func(x com.In) (err error) {
		switch x.T {
		case api.TerminateSession:
			dat := api.Unwrap[api.TerminateSessionRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleTerminateSession(*dat, h)
		case api.WebrtcInit:
			var out com.Out
			if dat := api.Unwrap[api.WebrtcInitRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleWebrtcInit(*dat, h, ap)
			}
			h.cord.Route(x, out)
		case api.WebrtcAnswer:
			dat := api.Unwrap[api.WebrtcAnswerRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcAnswer(*dat, h)
		case api.WebrtcIceCandidate:
			dat := api.Unwrap[api.WebrtcIceCandidateRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcIceCandidate(*dat, h)
		case api.StartGame:
			var out com.Out
			if dat := api.Unwrap[api.StartGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleGameStart(*dat, h)
			}
			h.cord.Route(x, out)
		case api.QuitGame:
			dat := api.Unwrap[api.GameQuitRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleQuitGame(*dat, h)
		case api.SaveGame:
			var out com.Out
			if dat := api.Unwrap[api.SaveGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleSaveGame(*dat, h)
			}
			h.cord.Route(x, out)
		case api.LoadGame:
			var out com.Out
			if dat := api.Unwrap[api.LoadGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleLoadGame(*dat, h)
			}
			h.cord.Route(x, out)
		case api.ChangePlayer:
			var out com.Out
			if dat := api.Unwrap[api.ChangePlayerRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleChangePlayer(*dat, h)
			}
			h.cord.Route(x, out)
		case api.ToggleMultitap:
			var out com.Out
			if dat := api.Unwrap[api.ToggleMultitapRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleToggleMultitap(*dat, h)
			}
			h.cord.Route(x, out)
		case api.RecordGame:
			var out com.Out
			if dat := api.Unwrap[api.RecordGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleRecordGame(*dat, h)
			}
			h.cord.Route(x, out)
		default:
			c.Log.Warn().Msgf("unhandled packet type %v", x.T)
		}
		return nil
	})
}

func (c *coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }
func (c *coordinator) CloseRoom(id string)    { c.Notify(api.CloseRoom, id) }
func (c *coordinator) IceCandidate(candidate string, sessionId network.Uid) {
	c.Notify(api.NewWebrtcIceCandidateRequest(sessionId, candidate))
}
