package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf worker.Config) (services service.Services) {
	httpSrv, err := NewHTTPServer(conf)
	if err != nil {
		panic("http failed: " + err.Error())
	}
	services.Add(httpSrv, NewHandler(conf, httpSrv.Addr))
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, "worker"))
	}
	return
}
