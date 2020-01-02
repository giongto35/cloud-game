package overlord

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

type Overlord struct {
	ctx context.Context
	cfg Config

	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg Config) *Overlord {
	return &Overlord{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.MonitoringConfig),
	}
}

func (o *Overlord) Run() error {
	go o.initializeOverlord()
	go o.RunMonitoringServer()
	return nil
}

func (o *Overlord) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overlord")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *Overlord) Shutdown() {
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

func makeHTTPServer(server *Server) *http.Server {
	r := mux.NewRouter()
	r.HandleFunc("/", server.GetWeb)
	r.HandleFunc("/ws", server.WS)
	r.HandleFunc("/wso", server.WSO)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))

	svmux := &http.ServeMux{}
	svmux.Handle("/", r)

	return makeServerFromMux(svmux)
}

func makeHTTPToHTTPSRedirectServer(server *Server) *http.Server {
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	}
	r := mux.NewRouter()
	r.HandleFunc("/", handleRedirect)
	r.HandleFunc("/ws", handleRedirect)
	r.HandleFunc("/wso", handleRedirect)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))

	svmux := &http.ServeMux{}
	svmux.Handle("/", r)

	return makeServerFromMux(svmux)
}

// initializeOverlord setup an overlord server
func (o *Overlord) initializeOverlord() {
	overlord := NewServer(o.cfg)

	var certManager *autocert.Manager
	var httpsSrv *http.Server

	log.Println("Initializing Overlord Server")
	if *config.Mode == config.ProdEnv || *config.Mode == config.StagingEnv {
		hostPolicy := func(ctx context.Context, host string) error {
			// Note: change to your real host

			allowedHost := "cloudretro.io"
			if host == allowedHost {
				return nil
			}
			return fmt.Errorf("acme/autocert: only %s host is allowed", allowedHost)
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

		httpsSrv = makeHTTPServer(overlord)
		httpsSrv.Addr = ":443"
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
		httpSrv = makeHTTPToHTTPSRedirectServer(overlord)
	} else {
		httpSrv = makeHTTPServer(overlord)
	}

	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = ":8000"
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}
