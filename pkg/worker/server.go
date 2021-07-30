package worker

import (
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
)

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

var echo = []byte{0x65, 0x63, 0x68, 0x6f}

func makeServerFromMux(mux *http.ServeMux) *http.Server {
	// set timeouts so that a slow or malicious client doesn't
	// hold resources forever
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
}

func makeHTTPServer() *http.Server {
	mux := &http.ServeMux{}
	mux.HandleFunc("/echo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(echo)
	})

	return makeServerFromMux(mux)
}

func makeHTTPToHTTPSRedirectServer() *http.Server {
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	}
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleRedirect)

	return makeServerFromMux(mux)
}

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
