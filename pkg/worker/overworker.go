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

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/config/worker"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/golang/glog"
)

type OverWorker struct {
	ctx context.Context
	cfg worker.Config

	monitoringServer *monitoring.ServerMonitoring
}

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

func New(ctx context.Context, cfg worker.Config) *OverWorker {
	return &OverWorker{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.MonitoringConfig),
	}
}

func (o *OverWorker) Run() error {
	go o.initializeWorker()
	go o.RunMonitoringServer()
	return nil
}

func (o *OverWorker) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overwork")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *OverWorker) Shutdown() {
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
		log.Println(w, "echo")
		http.Redirect(w, r, newURI, http.StatusFound)
	}
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleRedirect)

	return makeServerFromMux(mux)
}

func (o *OverWorker) spawnServer(port int) {
	var certManager *autocert.Manager
	var httpsSrv *http.Server

	if *config.Mode == config.ProdEnv || *config.Mode == config.StagingEnv {
		hostPolicy := func(ctx context.Context, host string) error {
			return nil
		}
		var leurl string
		if *config.Mode == config.StagingEnv {
			leurl = stagingLEURL
		} else {
			leurl = acme.LetsEncryptURL
		}

		certManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      autocert.DirCache("assets/cache"),
			Client:     &acme.Client{DirectoryURL: leurl},
		}

		httpsSrv = makeHTTPServer()
		httpsSrv.Addr = ":" + strconv.Itoa(port-9000+443) // equivalent https port
		httpsSrv.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}

		go func() {
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			err := httpsSrv.ListenAndServeTLS("", "")
			if err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	}

	var httpSrv *http.Server
	if *config.Mode == config.ProdEnv || *config.Mode == config.StagingEnv {
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
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

// initializeWorker setup a worker
func (o *OverWorker) initializeWorker() {
	worker := NewHandler(o.cfg)

	defer func() {
		log.Println("Close worker")
		worker.Close()
	}()

	go worker.Run()
	port := 9000
	// It's recommend to run one worker on one instance. This logic is to make sure more than 1 workers still work
	for {
		log.Println("Listening at port: localhost:", port)
		// err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			port++
			continue
		}
		if port == 9100 {
			// Cannot find port
			return
		}

		l.Close()

		o.spawnServer(port)
	}
}
