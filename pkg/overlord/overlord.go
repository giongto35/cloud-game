package overlord

import (
	"context"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/golang/glog"
)

type Overlord struct {
	ctx context.Context
	cfg Config

	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg Config) *Overlord {
	return &Overlord{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.MonitoringConfig),
	}
}

func (o *Overlord) Run() error {
	go o.initializeOverlord()
	go o.RunMonitoringServer()
	return nil
}

func (o *Overlord) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overlord")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *Overlord) Shutdown() {
	if err := o.monitoringServer.Shutdown(o.ctx); err != nil {
		glog.Errorln("Failed to shutdown monitoring server")
	}
}

// initializeOverlord setup an overlord server
func (o *Overlord) initializeOverlord() {
	overlord := NewServer()

	http.HandleFunc("/", overlord.GetWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))

	// browser facing port
	go func() {
		http.HandleFunc("/ws", overlord.WS)
	}()

	// worker facing port
	http.HandleFunc("/wso", overlord.WSO)
	log.Println("Listening at port: localhost:8000")
	err := http.ListenAndServe(":8000", nil)
	// Print err if overlord cannot launch
	if err != nil {
		log.Fatal(err)
	}
}
