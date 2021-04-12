package worker

import (
	"crypto/tls"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

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
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		log.Println(w, "echo")
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

func (wrk *Worker) spawnServer(address string) {
	var certManager *autocert.Manager
	var httpsSrv *http.Server

	mode := wrk.conf.Environment.Get()
	if mode.AnyOf(environment.Production, environment.Staging) {
		serverConfig := wrk.conf.Worker.Server
		httpsSrv = makeHTTPServer()
		httpsSrv.Addr = serverConfig.HttpsAddress

		if serverConfig.HttpsChain == "" || serverConfig.HttpsKey == "" {
			serverConfig.HttpsChain = ""
			serverConfig.HttpsKey = ""

			var leurl string
			if mode == environment.Staging {
				leurl = stagingLEURL
			} else {
				leurl = acme.LetsEncryptURL
			}

			certManager = &autocert.Manager{
				Prompt: autocert.AcceptTOS,
				Cache:  autocert.DirCache("assets/cache"),
				Client: &acme.Client{DirectoryURL: leurl},
			}

			httpsSrv.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
		}

		go func(chain string, key string) {
			log.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			err := httpsSrv.ListenAndServeTLS(chain, key)
			if err != nil {
				log.Printf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}(serverConfig.HttpsChain, serverConfig.HttpsKey)
	}

	var httpSrv *http.Server
	if mode.AnyOf(environment.Production, environment.Staging) {
		httpSrv = makeHTTPToHTTPSRedirectServer()
	} else {
		httpSrv = makeHTTPServer()
	}

	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}

	// ":3833" -> 3833
	port := strings.Split(address, ":")[0]
	if start, err := strconv.Atoi(port); err != nil {
		startServer(httpSrv, start)
	} else {
		log.Printf("error: couldn't extract port from %v", address)
	}
}

func startServer(serv *http.Server, startPort int) {
	// It's recommend to run one worker on one instance.
	// This logic is to make sure more than 1 workers still work
	for port, n := startPort, startPort+100; port < n; port++ {
		serv.Addr = ":" + strconv.Itoa(port)
		err := serv.ListenAndServe()
		switch err {
		case http.ErrServerClosed:
			log.Printf("HTTP(S) server was closed")
			return
		default:
		}
		port++

		if port == n {
			log.Printf("error: couldn't find an open port in range %v-%v\n", startPort, port)
		}
	}
}
