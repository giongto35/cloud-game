package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

// !to add proper shutdown on app termination with cancellation ctx

func New(conf worker.Config) service.Services {
	services := service.Services{}
	services.Add(NewHTTPServer(conf), NewHandler(conf))
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, "worker"))
	}
	return services
}
