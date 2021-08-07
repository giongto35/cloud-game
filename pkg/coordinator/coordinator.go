package coordinator

import (
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config) (services service.Group) {
	srv := NewServer(conf, games.NewLibWhitelisted(conf.Coordinator.Library, conf.Emulator))
	httpSrv, err := NewHTTPServer(conf, func(mux *http.ServeMux) {
		mux.HandleFunc("/ws", srv.WS)
		mux.HandleFunc("/wso", srv.WSO)
	})
	if err != nil {
		log.Fatalf("http init fail: %v", err)
	}
	services.Add(srv, httpSrv)
	if conf.Coordinator.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Coordinator.Monitoring, httpSrv.GetHost(), "cord"))
	}
	return
}
