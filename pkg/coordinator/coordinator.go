package coordinator

import (
	"html/template"
	"net/http"

	"github.com/giongto35/cloud-game/v3/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v3/pkg/config/shared"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/service"
)

func New(conf coordinator.Config, log *logger.Logger) (services service.Group) {
	lib := games.NewLibWhitelisted(conf.Coordinator.Library, conf.Emulator, log)
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

func NewHTTPServer(conf coordinator.Config, log *logger.Logger, fnMux func(*httpx.Mux) *httpx.Mux) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(s *httpx.Server) httpx.Handler {
			return fnMux(s.Mux().
				Handle("/", index(conf, log)).
				Static("/static/", "./web"))
		},
		httpx.WithServerConfig(conf.Coordinator.Server),
		httpx.WithLogger(log),
	)
}

func index(conf coordinator.Config, log *logger.Logger) httpx.Handler {
	const indexHTML = "./web/index.html"

	handler := func(tpl *template.Template, w httpx.ResponseWriter, r *httpx.Request) {
		if r.URL.Path != "/" {
			httpx.NotFound(w)
			return
		}
		// render index page with some tpl values
		tplData := struct {
			Analytics coordinator.Analytics
			Recording shared.Recording
		}{conf.Coordinator.Analytics, conf.Recording}
		if err := tpl.Execute(w, tplData); err != nil {
			log.Fatal().Err(err).Msg("error with the analytics template file")
		}
	}

	if conf.Coordinator.Debug {
		log.Info().Msgf("Using auto-reloading index.html")
		return httpx.HandlerFunc(func(w httpx.ResponseWriter, r *httpx.Request) {
			tpl, _ := template.ParseFiles(indexHTML)
			handler(tpl, w, r)
		})
	}

	indexTpl, err := template.ParseFiles(indexHTML)
	if err != nil {
		log.Fatal().Err(err).Msg("error with the HTML index page")
	}

	return httpx.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handler(indexTpl, writer, request)
	})
}
