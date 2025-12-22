package worker

import (
	"net/url"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
)

type Connection interface {
	Disconnect()
	Id() com.Uid
	ProcessPackets(func(api.In[com.Uid]) error) chan struct{}
	SetErrorHandler(func(error))

	Send(api.PT, any) ([]byte, error)
	Notify(api.PT, any)
	Route(api.In[com.Uid], *api.Out)
}

type coordinator struct {
	Connection
	log *logger.Logger
}

var connector com.Client

func newCoordinatorConnection(host string, conf config.Worker, addr string, log *logger.Logger) (*coordinator, error) {
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

	clog := log.Extend(log.With().Str(logger.ClientField, "c"))
	client := com.NewConnection[api.PT, api.In[com.Uid], api.Out, *api.Out](conn, id, clog)

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

	return c.ProcessPackets(func(x api.In[com.Uid]) (err error) {
		var out api.Out

		switch x.T {
		case api.WebrtcInit:
			err = api.Do(x, func(d api.WebrtcInitRequest[com.Uid]) { out = c.HandleWebrtcInit(d, w, ap) })
		case api.StartGame:
			err = api.Do(x, func(d api.StartGameRequest[com.Uid]) { out = c.HandleGameStart(d, w) })
		case api.SaveGame:
			err = api.Do(x, func(d api.SaveGameRequest[com.Uid]) { out = c.HandleSaveGame(d, w) })
		case api.LoadGame:
			err = api.Do(x, func(d api.LoadGameRequest[com.Uid]) { out = c.HandleLoadGame(d, w) })
		case api.ChangePlayer:
			err = api.Do(x, func(d api.ChangePlayerRequest[com.Uid]) { out = c.HandleChangePlayer(d, w) })
		case api.RecordGame:
			err = api.Do(x, func(d api.RecordGameRequest[com.Uid]) { out = c.HandleRecordGame(d, w) })
		case api.WebrtcAnswer:
			err = api.Do(x, func(d api.WebrtcAnswerRequest[com.Uid]) { c.HandleWebrtcAnswer(d, w) })
		case api.WebrtcIce:
			err = api.Do(x, func(d api.WebrtcIceCandidateRequest[com.Uid]) { c.HandleWebrtcIceCandidate(d, w) })
		case api.TerminateSession:
			err = api.Do(x, func(d api.TerminateSessionRequest[com.Uid]) { c.HandleTerminateSession(d, w) })
		case api.QuitGame:
			err = api.Do(x, func(d api.GameQuitRequest[com.Uid]) { c.HandleQuitGame(d, w) })
		case api.ResetGame:
			err = api.Do(x, func(d api.ResetGameRequest[com.Uid]) { c.HandleResetGame(d, w) })
		default:
			c.log.Warn().Msgf("unhandled packet type %v", x.T)
		}

		if out != (api.Out{}) {
			w.cord.Route(x, &out)
		}
		return
	})
}

func (c *coordinator) RegisterRoom(id string) { c.Notify(api.RegisterRoom, id) }

// CloseRoom sends a signal to coordinator which will remove that room from its list.
func (c *coordinator) CloseRoom(id string) { c.Notify(api.CloseRoom, id) }
func (c *coordinator) IceCandidate(candidate string, sessionId com.Uid) {
	c.Notify(api.WebrtcIce, api.WebrtcIceCandidateRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: sessionId}, Candidate: candidate})
}

func (c *coordinator) SendLibrary(w *Worker) {
	g := w.lib.GetAll()

	var gg = make([]api.GameInfo, len(g))
	for i, g := range g {
		gg[i] = api.GameInfo(g)
	}

	c.Notify(api.LibNewGameList, api.LibGameListInfo{T: 1, List: gg})
}

func (c *coordinator) SendPrevSessions(w *Worker) {
	sessions := w.lib.Sessions()

	// extract ids from save states, i.e. sessions
	var ids []string

	for _, id := range sessions {
		x, _ := api.ExplodeDeepLink(id)
		ids = append(ids, x)
	}

	c.Notify(api.PrevSessions, api.PrevSessionInfo{List: ids})
}
