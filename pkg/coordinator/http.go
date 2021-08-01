package coordinator

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

type HTTPServer struct {
	service.RunnableService

	server *httpx.Server
}

func NewHTTPServer(conf coordinator.Config, fnMux func(mux *http.ServeMux)) HTTPServer {
	return HTTPServer{server: httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(conf))
			h.Handle("/static/", static("./web"))
			fnMux(h)
			return h
		},
		httpx.WithServerConfig(conf.Coordinator.Server),
	)}
}

func (s HTTPServer) Run() { go s.server.Start() }

func (s HTTPServer) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }

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
		if err = tpl.Execute(w, conf.Coordinator.Analytics); err != nil {
			log.Fatal(err)
		}
	})
}

func static(dir string) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
}
