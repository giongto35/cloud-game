package coordinator

import (
	"html/template"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
)

func NewHTTPServer(conf coordinator.Config, fnMux func(mux *http.ServeMux)) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(conf))
			h.Handle("/static/", static("./web"))
			fnMux(h)
			return h
		},
		httpx.WithServerConfig(conf.Coordinator.Server),
	)
}

func index(conf coordinator.Config) http.Handler {
	tpl, err := template.ParseFiles("./web/index.html")
	if err != nil {
		log.Fatal(err)
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
			log.Fatal(err)
		}
	})
}

func static(dir string) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
}
