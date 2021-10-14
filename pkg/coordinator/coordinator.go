package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config, log *logger.Logger) (services service.Group) {
	hub := NewHub(conf, games.NewLibWhitelisted(conf.Coordinator.Library, conf.Emulator, log), log)
	httpSrv, err := NewHTTPServer(conf, log, func(mux *http.ServeMux) {
		mux.HandleFunc("/ws", hub.handleWebsocketUserConnection)
		mux.HandleFunc("/wso", hub.handleWebsocketWorkerConnection)
	})
	if err != nil {
		log.Error().Err(err).Msg("http server init fail")
		return
	}
	services.Add(hub, httpSrv)
	if conf.Coordinator.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Coordinator.Monitoring, httpSrv.GetHost(), log))
	}
	return
}
