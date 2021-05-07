package coordinator

import (
	"context"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

type HTTPServer struct {
	service.Service

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

func (s HTTPServer) Run() {
	go s.server.Start()
}

func (s HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
