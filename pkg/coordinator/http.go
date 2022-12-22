package coordinator

import (
	"html/template"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
)

func NewHTTPServer(conf coordinator.Config, log *logger.Logger, fnMux func(mux *http.ServeMux)) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(conf, log))
			h.Handle("/static/", static("./web"))
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

func static(dir string) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
}
