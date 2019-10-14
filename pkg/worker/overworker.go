package worker

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	workercfg "github.com/giongto35/cloud-game/pkg/worker/config"

	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/golang/glog"
)

type OverWorker struct {
	ctx context.Context
	cfg workercfg.Config

	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg workercfg.Config) *OverWorker {
	return &OverWorker{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.MonitoringConfig),
	}
}

func (o *OverWorker) Run() error {
	go o.initializeWorker()
	go o.RunMonitoringServer()
	return nil
}

func (o *OverWorker) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overwork")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *OverWorker) Shutdown() {
	if err := o.monitoringServer.Shutdown(o.ctx); err != nil {
		glog.Errorln("Failed to shutdown monitoring server")
	}
}

// initializeWorker setup a worker
func (o *OverWorker) initializeWorker() {
	worker := NewHandler(o.cfg)

	defer func() {
		log.Println("Close worker")
		worker.Close()
	}()

	go worker.Run()
	port := 9000
	// It's recommend to run one worker on one instance. This logic is to make sure more than 1 workers still work
	for {
		log.Println("Listening at port: localhost:", port)
		// err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			port++
			continue
		}
		if port == 9100 {
			// Cannot find port
			return
		}

		l.Close()

		// echo endpoint is where user will request to test latency
		http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			fmt.Fprintf(w, "echo")
		})

		http.ListenAndServe(":"+strconv.Itoa(port), nil)
	}
}
