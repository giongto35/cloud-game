package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config) (services service.Services) {
	srv := NewServer(conf, games.NewLibWhitelisted(conf.Coordinator.Library, conf.Emulator))
	services.Add(
		srv,
		NewHTTPServer(conf, func(mux *http.ServeMux) {
			mux.HandleFunc("/ws", srv.WS)
			mux.HandleFunc("/wso", srv.WSO)
		}))
	if conf.Coordinator.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Coordinator.Monitoring, "cord"))
	}
	return
}
