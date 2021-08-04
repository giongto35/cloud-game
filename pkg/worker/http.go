package worker

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
)

func NewHTTPServer(conf worker.Config) (*httpx.Server, error) {
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
	)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
