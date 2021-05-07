package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config) service.Services {
	lib := getLibrary(conf)
	lib.Scan()

	hub := NewHub(conf, lib)

	services := service.Services{}
	services.Add(NewHTTPServer(conf, func(mux *http.ServeMux) {
		mux.HandleFunc("/ws", hub.handleNewWebsocketUserConnection)
		mux.HandleFunc("/wso", hub.handleNewWebsocketWorkerConnection)
	}))
	if conf.Coordinator.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Coordinator.Monitoring, "cord"))
	}
	return services
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
