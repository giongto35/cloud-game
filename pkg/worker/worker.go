package worker

import (
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config/worker"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/service"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro/manager/remotehttp"
)

type Worker struct {
	address string
	conf    worker.Config
	cord    *coordinator
	log     *logger.Logger
	router  Router
	storage CloudStorage
	done    chan struct{}
}

const retry = 10 * time.Second

func New(conf worker.Config, log *logger.Logger, done chan struct{}) (services service.Group) {
	if err := remotehttp.CheckCores(conf.Emulator, log); err != nil {
		log.Error().Err(err).Msg("cores sync error")
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
		// no need just for one route
		httpx.HttpsRedirect(false),
		httpx.WithPortRoll(true),
		httpx.WithZone(conf.Worker.Network.Zone),
		httpx.WithLogger(log),
	)
	if err != nil {
		log.Error().Err(err).Msg("http init fail")
		return
	}
	services.Add(h)
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, h.GetHost(), log))
	}
	st, err := GetCloudStorage(conf.Storage.Provider, conf.Storage.Key)
	if err != nil {
		log.Error().Err(err).Msgf("cloud storage fail, using dummy cloud storage instead")
	}
	services.Add(&Worker{address: h.Addr, conf: conf, done: done, log: log, storage: st, router: NewRouter()})

	return
}

func (w *Worker) Run() {
	go func() {
		remoteAddr := w.conf.Worker.Network.CoordinatorAddress
		defer func() {
			if w.cord != nil {
				w.cord.Disconnect()
			}
			w.router.Close()
			w.log.Debug().Msgf("Service loop end")
		}()

		for {
			select {
			case <-w.done:
				return
			default:
				cord, err := newCoordinatorConnection(remoteAddr, w.conf.Worker, w.address, w.log)
				if err != nil {
					w.log.Error().Err(err).Msgf("no connection: %v. Retrying in %v", remoteAddr, retry)
					time.Sleep(retry)
					continue
				}
				w.cord = cord
				w.cord.log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
				<-w.cord.HandleRequests(w)
				w.router.Close()
			}
		}
	}()
}
func (w *Worker) Stop() error { return nil }
