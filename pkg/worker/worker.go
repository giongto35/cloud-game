package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
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

	if err := remotehttp.CheckCores(conf.Emulator, log); err != nil {
		log.Error().Err(err).Msg("cores sync error")
	}

	mainHandler := NewHandler(httpSrv.Addr, conf, log)

	services.Add(httpSrv, mainHandler)
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, httpSrv.GetHost(), log))
	}
	return
}
