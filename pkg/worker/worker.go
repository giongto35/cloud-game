package worker

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf worker.Config) (services service.Group) {
	httpSrv, err := NewHTTPServer(conf)
	if err != nil {
		log.Fatalf("http init fail: %v", err)
	}
	services.Add(
		httpSrv,
		NewHandler(conf, httpSrv.Addr),
	)
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, httpSrv.GetHost(), "worker"))
	}
	return
}
