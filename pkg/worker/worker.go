package worker

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/lock"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/server"
)

type Worker struct {
	conf     worker.Config
	ctx      context.Context
	services *server.Services

	// to pause initialization
	lock *lock.TimeLock
}

func New(ctx context.Context, conf worker.Config) *Worker {
	services := server.Services{}
	return &Worker{
		ctx:  ctx,
		conf: conf,
		services: services.AddIf(
			conf.Worker.Monitoring.IsEnabled(), monitoring.New(conf.Worker.Monitoring, "worker"),
		),
		lock: lock.NewLock(),
	}
}

// !to add proper shutdown on app termination with cancellation ctx

func (w *Worker) Run() {
	go func() {
		h := NewHandler(w)
		defer func() {
			log.Printf("[worker] Closing handler")
			h.Close()
		}()

		go h.Run()
		if !w.conf.Loaded {
			w.lock.LockFor(time.Second * 10)
			h.RequestConfig()
		}
		h.Prepare()
		w.init()
	}()
	w.services.Start()
}

func (w *Worker) init() {
	conf := w.conf.Worker
	httpx.NewServer(
		conf.Server.GetAddr(),
		func(_ *httpx.Server) http.Handler {
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
}

func (w *Worker) Shutdown() { w.services.Shutdown(w.ctx) }
