package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf worker.Config, log *logger.Logger) (services service.Group) {
	httpSrv, err := NewHTTPServer(conf, log)
	if err != nil {
		log.Error().Err(err).Msg("http init fail")
		return
	}

	mainHandler := NewHandler(conf, httpSrv.Addr, log)
	mainHandler.Prepare()

	services.Add(httpSrv)
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, httpSrv.GetHost(), log))
	}
	services.Add(mainHandler)
	return
}
