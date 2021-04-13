package worker

import (
	"context"
	"log"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/lock"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
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
		wrk.spawnServer(wrk.conf.Worker.Server.Address)
	}()
	wrk.services.Start()
}

func (wrk *Worker) Shutdown() { wrk.services.Shutdown(wrk.ctx) }
