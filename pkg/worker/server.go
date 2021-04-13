package worker

import (
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
)

func (wrk *Worker) spawnServer(addr string) {
	conf := wrk.conf.Worker

	address := network.Address(addr)
	if conf.Server.Https {
		address = network.Address(conf.Server.Tls.Address)
	}

	httpx.NewServer(
		string(address),
		func(serv *httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				log.Println(w, "echo")
			})
			return h
		},
		httpx.WithServerConfig(conf.Server),
		httpx.WithPortRoll(true),
	).Start()
}
