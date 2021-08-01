package worker

import (
	"context"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

type HTTPServer struct {
	service.RunnableService

	server *httpx.Server
}

func NewHTTPServer(conf worker.Config) HTTPServer {
	return HTTPServer{server: httpx.NewServer(
		conf.Worker.Server.GetAddr(),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.HandleFunc(conf.Worker.Network.PingEndpoint, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				_, _ = w.Write([]byte{0x65, 0x63, 0x68, 0x6f}) // echo
			})
			return h
		},
		httpx.WithServerConfig(conf.Worker.Server),
		// no need just for one route
		httpx.HttpsRedirect(false),
		httpx.WithPortRoll(true),
	)}
}

func (s HTTPServer) Run() { go s.server.Start() }

func (s HTTPServer) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
