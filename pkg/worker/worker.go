package worker

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/lock"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/server"
)

type Worker struct {
	ctx      context.Context
	conf     worker.Config
	services server.Services
	// to pause initialization
	lock *lock.TimeLock
}

func New(ctx context.Context, conf worker.Config) *Worker {
	return &Worker{
		ctx:  ctx,
		conf: conf,
		services: []server.Server{
			monitoring.NewServerMonitoring(conf.Worker.Monitoring, "worker"),
		},
		lock: lock.NewLock(),
	}
}

func (wrk *Worker) Run() {
	go func() {
		h := NewHandler(wrk.conf, wrk)
		defer func() {
			log.Printf("[worker] Closing handler")
			h.Close()
		}()

		go h.Run()
		if !wrk.conf.Loaded {
			wrk.lock.LockFor(time.Second * 10)
			h.RequestConfig()
		}
		h.Prepare()
		wrk.init()
	}()
	wrk.services.Start()
}

func (wrk *Worker) init() {
	conf := wrk.conf.Worker

	address := network.Address(conf.Server.Address)
	if conf.Server.Https {
		address = network.Address(conf.Server.Tls.Address)
	}

	httpx.NewServer(
		string(address),
		func(serv *httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				log.Println(w, "echo")
			})
			return h
		},
		httpx.WithServerConfig(conf.Server),
		httpx.WithPortRoll(true),
	).Start()
}

func (wrk *Worker) Shutdown() { wrk.services.Shutdown(wrk.ctx) }
