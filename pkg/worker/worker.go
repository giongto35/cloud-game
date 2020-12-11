package worker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/golang/glog"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type Worker struct {
	ctx context.Context
	cfg worker.Config

	monitoringServer *monitoring.ServerMonitoring
}

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

func New(ctx context.Context, cfg worker.Config) *Worker {
	return &Worker{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.Worker.Monitoring),
	}
}

func (o *Worker) Run() error {
	go o.initializeWorker()
	go o.RunMonitoringServer()
	return nil
}

func (o *Worker) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overwork")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *Worker) Shutdown() {
	if err := o.monitoringServer.Shutdown(o.ctx); err != nil {
		glog.Errorln("Failed to shutdown monitoring server")
	}
}

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

func (o *Worker) spawnServer(port int) {
	var certManager *autocert.Manager
	var httpsSrv *http.Server

	mode := o.cfg.Environment.Mode
	if mode.AnyOf(environment.Production, environment.Staging) {
		serverConfig := o.cfg.Server
		httpsSrv = makeHTTPServer()
		httpsSrv.Addr = fmt.Sprintf(":%d", serverConfig.HttpsPort)

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
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
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

	httpSrv.Addr = ":" + strconv.Itoa(port)
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Printf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

// initializeWorker setup a worker
func (o *Worker) initializeWorker() {
	wrk := NewHandler(o.cfg)

	defer func() {
		log.Println("Close worker")
		wrk.Close()
	}()

	go wrk.Run()
	port := o.cfg.Server.Port
	// It's recommend to run one worker on one instance.
	// This logic is to make sure more than 1 workers still work
	portsNum := 100
	for {
		portsNum--
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			port++
			continue
		}

		if portsNum < 1 {
			log.Printf("Couldn't find an open port in range %v-%v\n", o.cfg.Server.Port, port)
			// Cannot find port
			return
		}

		_ = l.Close()

		log.Printf("Worker port is %v", port)

		o.spawnServer(port)
	}
}
