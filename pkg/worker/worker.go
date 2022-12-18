package worker

import (
	"context"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/worker/storage"
)

type Worker struct {
	address string
	conf    worker.Config
	cord    *coordinator
	ctx     context.Context
	log     *logger.Logger
	router  Router
	storage storage.CloudStorage
}

const retry = 10 * time.Second

func New(ctx context.Context, conf worker.Config, log *logger.Logger) (services service.Group) {
	if err := remotehttp.CheckCores(conf.Emulator, log); err != nil {
		log.Error().Err(err).Msg("cores sync error")
	}
	h, err := httpx.NewServer(
		conf.Worker.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.HandleFunc(conf.Worker.Network.PingEndpoint, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				_, _ = w.Write([]byte{0x65, 0x63, 0x68, 0x6f}) // echo
			})
			return h
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
	services.Add(&Worker{
		address: h.Addr,
		conf:    conf,
		ctx:     ctx,
		log:     log,
		storage: storage.GetCloudStorage(conf.Storage.Provider, conf.Storage.Key),
		router:  NewRouter(),
	})

	return
}

func (w *Worker) Run() {
	go func() {
		remoteAddr := w.conf.Worker.Network.CoordinatorAddress
		defer func() {
			if w.cord != nil {
				w.cord.Close()
			}
			w.router.Close()
			w.log.Debug().Msgf("Service loop end")
		}()
		for {
			select {
			case <-w.ctx.Done():
				return
			default:
				conn, err := connect(remoteAddr, w.conf.Worker, w.address, w.log)
				if err != nil {
					w.log.Error().Err(err).Msgf("no connection: %v. Retrying in %v", remoteAddr, retry)
					time.Sleep(retry)
					continue
				}
				w.cord = conn
				w.cord.Log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
				w.cord.HandleRequests(w)
				<-w.cord.Done()
				w.router.Close()
			}
		}
	}()
}
func (w *Worker) Shutdown(context.Context) error { return nil }
