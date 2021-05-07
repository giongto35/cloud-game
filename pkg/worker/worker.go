package worker

import (
	"context"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/server"
)

type Worker struct {
	conf     worker.Config
	ctx      context.Context
	services *server.Services
}

func New(ctx context.Context, conf worker.Config) *Worker {
	services := server.Services{}
	return &Worker{
		ctx:  ctx,
		conf: conf,
		services: services.AddIf(
			conf.Worker.Monitoring.IsEnabled(), monitoring.New(conf.Worker.Monitoring, "worker"),
		),
	}
}

// !to add proper shutdown on app termination with cancellation ctx

func (w *Worker) Run(ctx context.Context) {
	conf := w.conf.Worker
	go NewHandler(w).Run(ctx)
	go httpx.NewServer(
		conf.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.HandleFunc(conf.Network.PingEndpoint, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				_, _ = w.Write([]byte{0x65, 0x63, 0x68, 0x6f}) // echo
			})
			return h
		},
		httpx.WithServerConfig(conf.Server),
		// no need just for one route
		httpx.HttpsRedirect(false),
		httpx.WithPortRoll(true),
	).Start()
	w.services.Start()
}

func (w *Worker) Shutdown() { w.services.Shutdown(w.ctx) }
