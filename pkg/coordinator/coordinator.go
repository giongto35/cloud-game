package coordinator

import (
	"html/template"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(conf coordinator.Config, log *logger.Logger) (services service.Group) {
	lib := games.NewLibWhitelisted(conf.Coordinator.Library, conf.Emulator, log)
	lib.Scan()
	hub := NewHub(conf, lib, log)
	h, err := NewHTTPServer(conf, log, func(mux *http.ServeMux) {
		mux.HandleFunc("/ws", hub.handleUserConnection)
		mux.HandleFunc("/wso", hub.handleWorkerConnection)
	})
	if err != nil {
		log.Error().Err(err).Msg("http server init fail")
		return
	}
	services.Add(hub, h)
	if conf.Coordinator.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Coordinator.Monitoring, h.GetHost(), log))
	}
	return
}

func NewHTTPServer(conf coordinator.Config, log *logger.Logger, fnMux func(*http.ServeMux)) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(conf, log))
			h.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))
			fnMux(h)
			return h
		},
		httpx.WithServerConfig(conf.Coordinator.Server),
		httpx.WithLogger(log),
	)
}

func index(conf coordinator.Config, log *logger.Logger) http.Handler {
	tpl, err := template.ParseFiles("./web/index.html")
	if err != nil {
		log.Fatal().Err(err).Msg("error with the HTML index page")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 on unknown
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// render index page with some tpl values
		tplData := struct {
			Analytics coordinator.Analytics
			Recording shared.Recording
		}{conf.Coordinator.Analytics, conf.Recording}
		if err = tpl.Execute(w, tplData); err != nil {
			log.Fatal().Err(err).Msg("error with the analytics template file")
		}
	})
}
