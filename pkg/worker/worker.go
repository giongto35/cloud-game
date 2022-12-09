package worker

import (
	"context"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(ctx context.Context, conf worker.Config, log *logger.Logger) (services service.Group) {
	if err := remotehttp.CheckCores(conf.Emulator, log); err != nil {
		log.Error().Err(err).Msg("cores sync error")
	}
	http, err := NewHTTPServer(conf, log)
	if err != nil {
		log.Error().Err(err).Msg("http init fail")
		return
	}
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, http.GetHost(), log))
	}
	main := NewHandler(ctx, http.Addr, conf, log)
	services.Add(http, main)
	return
}
