package coordinator

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/service"
)

func New(conf config.CoordinatorConfig, log *logger.Logger) (services service.Group) {
	lib := games.NewLib(conf.Coordinator.Library, conf.Emulator, log)
	lib.Scan()
	hub := NewHub(conf, lib, log)
	h, err := NewHTTPServer(conf, log, func(mux *httpx.Mux) *httpx.Mux {
		mux.HandleFunc("/ws", hub.handleUserConnection())
		mux.HandleFunc("/wso", hub.handleWorkerConnection())
		return mux
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

func NewHTTPServer(conf config.CoordinatorConfig, log *logger.Logger, fnMux func(*httpx.Mux) *httpx.Mux) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(s *httpx.Server) httpx.Handler { return fnMux(s.Mux().Handle("/", index(conf, log))) },
		httpx.WithServerConfig(conf.Coordinator.Server),
		httpx.WithLogger(log),
	)
}

func index(conf config.CoordinatorConfig, log *logger.Logger) httpx.Handler {
	const indexHTML = "./web/index.html"

	indexTpl := template.Must(template.ParseFiles(indexHTML))

	// render index page with some tpl values
	tplData := struct {
		Analytics config.Analytics
		Recording config.Recording
	}{conf.Coordinator.Analytics, conf.Recording}

	handler := func(tpl *template.Template, w httpx.ResponseWriter, r *httpx.Request) {
		if err := tpl.Execute(w, tplData); err != nil {
			log.Fatal().Err(err).Msg("error with the analytics template file")
		}
	}

	h := httpx.FileServer("./web")

	if conf.Coordinator.Debug {
		log.Info().Msgf("Using auto-reloading index.html")
		return httpx.HandlerFunc(func(w httpx.ResponseWriter, r *httpx.Request) {
			if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/index.html") {
				tpl := template.Must(template.ParseFiles(indexHTML))
				handler(tpl, w, r)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	return httpx.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/index.html") {
			handler(indexTpl, w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
