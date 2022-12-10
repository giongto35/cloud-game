package worker

import (
	"context"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

func New(ctx context.Context, conf worker.Config, log *logger.Logger) (services service.Group) {
	h, err := NewHTTPServer(conf, log)
	if err != nil {
		log.Error().Err(err).Msg("http init fail")
		return
	}
	if conf.Worker.Monitoring.IsEnabled() {
		services.Add(monitoring.New(conf.Worker.Monitoring, h.GetHost(), log))
	}
	services.Add(h, NewWorkerService(ctx, h.Addr, conf, log))
	return
}

func NewHTTPServer(conf worker.Config, log *logger.Logger) (*httpx.Server, error) {
	srv, err := httpx.NewServer(
		conf.Worker.GetAddr(),
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
		httpx.WithZone(conf.Worker.Network.Zone),
		httpx.WithLogger(log),
	)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
