package worker

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/cloud"
	"github.com/giongto35/cloud-game/v3/pkg/worker/room"
)

type Worker struct {
	address  string
	conf     config.WorkerConfig
	cord     *coordinator
	lib      games.GameLibrary
	launcher games.Launcher
	log      *logger.Logger
	mana     *caged.Manager
	router   *room.GameRouter
	services [2]interface {
		Run()
		Stop() error
	}
	storage cloud.Storage

	// the WebRTC API (and its single UDP port) lives as long as the worker:
	// recreating it on every coordinator reconnect leaked the port and moved
	// the media to the next (unpublished) one
	rtcOnce    sync.Once
	rtcFactory *webrtc.ApiFactory
	rtcErr     error
}

func (w *Worker) webrtcFactory() (*webrtc.ApiFactory, error) {
	w.rtcOnce.Do(func() { w.rtcFactory, w.rtcErr = webrtc.NewApiFactory(w.conf.Webrtc, w.log, nil) })
	return w.rtcFactory, w.rtcErr
}

func New(conf config.WorkerConfig, log *logger.Logger) (*Worker, error) {
	manager := caged.NewManager(log)
	if err := manager.Load(caged.Libretro, conf); err != nil {
		return nil, fmt.Errorf("couldn't cage libretro: %v", err)
	}

	library := games.NewLib(conf.Library, conf.Emulator, log)
	library.Scan()

	worker := &Worker{
		conf:     conf,
		lib:      library,
		launcher: games.NewGameLauncher(library),
		log:      log,
		mana:     manager,
		router:   room.NewGameRouter(),
	}

	h, err := httpx.NewServer(
		conf.Worker.GetAddr(),
		func(s *httpx.Server) httpx.Handler {
			return s.Mux().HandleW(conf.Worker.Network.PingEndpoint, func(w httpx.ResponseWriter) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				_, _ = w.Write([]byte{0x65, 0x63, 0x68, 0x6f}) // echo
			})
		},
		httpx.WithServerConfig(conf.Worker.Server),
		httpx.HttpsRedirect(false),
		httpx.WithPortRoll(true),
		httpx.WithZone(conf.Worker.Network.Zone),
		httpx.WithLogger(log),
	)
	if err != nil {
		return nil, fmt.Errorf("http init fail: %w", err)
	}
	worker.address = h.Addr
	worker.services[0] = h
	if conf.Worker.Monitoring.IsEnabled() {
		worker.services[1] = monitoring.New(conf.Worker.Monitoring, h.GetHost(), log)
	}
	st, err := cloud.Store(conf.Storage, log)
	if err != nil {
		log.Warn().Err(err).Msgf("cloud storage fail, using no storage")
	}
	worker.storage = st

	return worker, nil
}

func (w *Worker) Reset() { w.router.Reset() }

func (w *Worker) Start(done chan struct{}) {
	for _, s := range w.services {
		if s != nil {
			s.Run()
		}
	}

	// !to restore alive worker info when coordinator connection was lost
	retryStep := 5 * time.Second
	onRetryFail := func(err error) {
		w.log.Warn().Err(err).Msgf("socket fail. Retrying in %v", retryStep)
		backoff(&retryStep, 1*time.Hour)
	}

	go func() {
		remoteAddr := w.conf.Worker.Network.CoordinatorAddress
		defer func() {
			if w.cord != nil {
				w.cord.Disconnect()
			}
			w.Reset()
		}()

		for {
			select {
			case <-done:
				return
			default:
				w.Reset()
				cord, err := newCoordinatorConnection(remoteAddr, w.conf.Worker, w.address, w.log)
				if err != nil {
					onRetryFail(err)
					continue
				}
				cord.SetErrorHandler(onRetryFail)
				w.cord = cord
				w.cord.log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
				wait := w.cord.HandleRequests(w)
				w.cord.SendLibrary(w)
				w.cord.SendPrevSessions(w)
				<-wait
				retryStep = 5 * time.Second
			}
		}
	}()
}

func (w *Worker) Stop() error {
	w.Reset()
	var err error
	for _, s := range w.services {
		if s != nil {
			err0 := s.Stop()
			err = errors.Join(err, err0)
		}
	}
	return err
}

// backoff doubles the sleep duration up to the max
func backoff(d *time.Duration, max time.Duration) { time.Sleep(*d); *d = min(*d*2, max) }
