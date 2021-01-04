package worker

import (
	"context"
	"log"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/lock"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/server"
	"github.com/golang/glog"
)

type Worker struct {
	ctx     context.Context
	conf    worker.Config
	servers []server.Server
	// to pause initialization
	lock *lock.TimeLock
}

func New(ctx context.Context, conf worker.Config) *Worker {
	return &Worker{ctx: ctx, conf: conf, lock: lock.NewLock()}
}

func (wrk *Worker) Run() {
	go wrk.init()
	wrk.servers = []server.Server{
		monitoring.NewServerMonitoring(wrk.conf.Worker.Monitoring, "worker"),
	}
	wrk.startModules()
}

func (wrk *Worker) init() {
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
	wrk.spawnServer(wrk.conf.Worker.Server.Port)
}

func (wrk *Worker) startModules() {
	glog.Info("[worker] active modules: ", wrk.servers)
	for _, s := range wrk.servers {
		s := s
		go func() {
			if err := s.Init(wrk.conf); err != nil {
				glog.Errorf("failed server init")
				return
			}
			if err := s.Run(); err != nil {
				glog.Errorf("failed start server")
			}
		}()
	}
}

// !to add a proper HTTP(S) server shutdown (cws/handler bad loop)
func (wrk *Worker) Shutdown() {
	for _, s := range wrk.servers {
		if err := s.Shutdown(wrk.ctx); err != nil {
			glog.Errorln("failed server shutdown")
		}
	}
}
