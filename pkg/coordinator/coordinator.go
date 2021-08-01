package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config) (services service.Services) {
	lib := getLibrary(conf)
	lib.Scan()

	srv := NewServer(conf, lib)

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

// !to rewrite
// getLibrary scans files with explicit extensions or
// the extensions specified in the emulator config.
func getLibrary(conf coordinator.Config) games.GameLibrary {
	if len(conf.Coordinator.Library.Supported) == 0 {
		conf.Coordinator.Library.Supported = conf.Emulator.GetSupportedExtensions()
	}
	return games.NewLibrary(conf.Coordinator.Library)
}
