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

func (c *coordinator) HandleRequests(s *Service) {
	ap, err := webrtc.NewApiFactory(s.conf.Webrtc, c.Log, nil)
	if err != nil {
		c.Log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}
	//
	c.ProcessMessages()
	//
	c.OnPacket(func(x com.In) (err error) {
		switch x.T {
		case api.WebrtcInit:
			var out com.Out
			if dat := api.Unwrap[api.WebrtcInitRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleWebrtcInit(*dat, s, ap)
			}
			s.cord.Route(x, out)
		case api.WebrtcAnswer:
			dat := api.Unwrap[api.WebrtcAnswerRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcAnswer(*dat, s)
		case api.WebrtcIceCandidate:
			dat := api.Unwrap[api.WebrtcIceCandidateRequest](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcIceCandidate(*dat, s)
		case api.StartGame:
			var out com.Out
			if dat := api.Unwrap[api.StartGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleGameStart(*dat, s)
			}
			s.cord.Route(x, out)
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
			var out com.Out
			if dat := api.Unwrap[api.SaveGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleSaveGame(*dat, s)
			}
			s.cord.Route(x, out)
		case api.LoadGame:
			var out com.Out
			if dat := api.Unwrap[api.LoadGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleLoadGame(*dat, s)
			}
			s.cord.Route(x, out)
		case api.ChangePlayer:
			var out com.Out
			if dat := api.Unwrap[api.ChangePlayerRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				out = c.HandleChangePlayer(*dat, s)
			}
			s.cord.Route(x, out)
		case api.ToggleMultitap:
			var out com.Out
			if dat := api.Unwrap[api.ToggleMultitapRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleToggleMultitap(*dat, s)
			}
			s.cord.Route(x, out)
		case api.RecordGame:
			var out com.Out
			if dat := api.Unwrap[api.RecordGameRequest](x.Payload); dat == nil {
				err, out = api.ErrMalformed, com.EmptyPacket
			} else {
				c.HandleRecordGame(*dat, s)
			}
			s.cord.Route(x, out)
		default:
			c.Log.Warn().Msgf("unhandled packet type %v", x.T)
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
