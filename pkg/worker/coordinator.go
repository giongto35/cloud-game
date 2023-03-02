package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
)

type Connection interface {
	Disconnect()
	Id() com.Uid
	Listen() chan struct{}
	OnPacket(func(api.In[com.Uid]) error)

	Send(api.PT, any) ([]byte, error)
	Notify(api.PT, any)
	Route(api.In[com.Uid], *api.Out)
}

type coordinator struct {
	Connection
	log *logger.Logger
}

var connector com.Client

func newCoordinatorConnection(host string, conf worker.Worker, addr string, log *logger.Logger) (*coordinator, error) {
	scheme := "ws"
	if conf.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Network.Endpoint}

	log.Debug().
		Str(logger.ClientField, "c").
		Str(logger.DirectionField, logger.MarkOut).
		Msgf("Handshake %s", address.String())

	id := com.NewUid()
	req, err := buildConnQuery(id, conf, addr)
	if req != "" && err == nil {
		address.RawQuery = "data=" + req
	} else {
		return nil, err
	}

	conn, err := connector.Connect(address)
	if err != nil {
		return nil, err
	}

	client := com.NewConnection[com.Uid, api.PT, api.In[com.Uid], api.Out](
		conn, id,
		log.Extend(log.With().Str(logger.ClientField, "c")),
	)
	return &coordinator{
		Connection: client,
		log:        log.Extend(log.With().Str("cid", client.Id().Short())),
	}, nil
}

func (c *coordinator) HandleRequests(w *Worker) chan struct{} {
	ap, err := webrtc.NewApiFactory(w.conf.Webrtc, c.log, nil)
	if err != nil {
		c.log.Panic().Err(err).Msg("WebRTC API creation has been failed")
	}
	skipped := api.Out{}

	c.OnPacket(func(x api.In[com.Uid]) (err error) {
		var out api.Out
		switch x.T {
		case api.WebrtcInit:
			if dat := api.Unwrap[api.WebrtcInitRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleWebrtcInit(*dat, w, ap)
			}
		case api.WebrtcAnswer:
			dat := api.Unwrap[api.WebrtcAnswerRequest[com.Uid]](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcAnswer(*dat, w)
		case api.WebrtcIce:
			dat := api.Unwrap[api.WebrtcIceCandidateRequest[com.Uid]](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleWebrtcIceCandidate(*dat, w)
		case api.StartGame:
			if dat := api.Unwrap[api.StartGameRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleGameStart(*dat, w)
			}
		case api.TerminateSession:
			dat := api.Unwrap[api.TerminateSessionRequest[com.Uid]](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleTerminateSession(*dat, w)
		case api.QuitGame:
			dat := api.Unwrap[api.GameQuitRequest[com.Uid]](x.Payload)
			if dat == nil {
				return api.ErrMalformed
			}
			c.HandleQuitGame(*dat, w)
		case api.SaveGame:
			if dat := api.Unwrap[api.SaveGameRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleSaveGame(*dat, w)
			}
		case api.LoadGame:
			if dat := api.Unwrap[api.LoadGameRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleLoadGame(*dat, w)
			}
		case api.ChangePlayer:
			if dat := api.Unwrap[api.ChangePlayerRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleChangePlayer(*dat, w)
			}
		case api.ToggleMultitap:
			if dat := api.Unwrap[api.ToggleMultitapRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				c.HandleToggleMultitap(*dat, w)
			}
		case api.RecordGame:
			if dat := api.Unwrap[api.RecordGameRequest[com.Uid]](x.Payload); dat == nil {
				err, out = api.ErrMalformed, api.EmptyPacket
			} else {
				out = c.HandleRecordGame(*dat, w)
			}
		default:
			c.log.Warn().Msgf("unhandled packet type %v", x.T)
		}
		if out != skipped {
			w.cord.Route(x, &out)
		}
		return err
	})
	return c.Listen()
}

func (c *coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }

// CloseRoom sends a signal to coordinator which will remove that room from its list.
func (c *coordinator) CloseRoom(id string) { c.Notify(api.CloseRoom, id) }
func (c *coordinator) IceCandidate(candidate string, sessionId com.Uid) {
	c.Notify(api.WebrtcIce, api.WebrtcIceCandidateRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: sessionId}, Candidate: candidate})
}
