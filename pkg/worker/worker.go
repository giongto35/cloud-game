package worker

import (
	"errors"
	"fmt"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/cloud"
	"github.com/giongto35/cloud-game/v3/pkg/worker/room"
)

type Worker struct {
	address  string
	conf     config.WorkerConfig
	cord     *coordinator
	log      *logger.Logger
	mana     *caged.Manager
	router   *room.GameRouter
	services [2]interface {
		Run()
		Stop() error
	}
	storage cloud.Storage
}

func (w *Worker) Reset() {
	w.log.Debug().Msgf("Users before close: %v", w.router.Users())
	w.router.Close()
	w.log.Debug().Msgf("Users after close: %v", w.router.Users())
}

const retry = 10 * time.Second

func New(conf config.WorkerConfig, log *logger.Logger) (*Worker, error) {
	manager := caged.NewManager(log)
	if err := manager.Load(caged.Libretro, conf); err != nil {
		return nil, fmt.Errorf("couldn't cage libretro: %v", err)
	}
	worker := &Worker{conf: conf, log: log, mana: manager, router: room.NewGameRouter()}

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
	st, err := cloud.Store(conf.Storage.Provider, conf.Storage.Key)
	if err != nil {
		log.Warn().Err(err).Msgf("cloud storage fail, using no storage")
	}
	worker.storage = st

	return worker, nil
}

func (w *Worker) Start(done chan struct{}) {
	for _, s := range w.services {
		if s != nil {
			s.Run()
		}
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
				cord, err := newCoordinatorConnection(remoteAddr, w.conf.Worker, w.address, w.log)
				if err != nil {
					w.log.Warn().Err(err).Msgf("no connection: %v. Retrying in %v", remoteAddr, retry)
					time.Sleep(retry)
					continue
				}
				w.cord = cord
				w.cord.log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
				<-w.cord.HandleRequests(w)
				w.Reset()
			}
		}
	}()
}

func (w *Worker) Stop() error {
	var err error
	for _, s := range w.services {
		if s != nil {
			err0 := s.Stop()
			err = errors.Join(err, err0)
		}
	}
	return err
}
