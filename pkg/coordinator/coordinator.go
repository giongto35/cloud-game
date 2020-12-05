package coordinator

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const stagingLEURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

type Coordinator struct {
	ctx context.Context
	cfg coordinator.Config

	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg coordinator.Config) *Coordinator {
	return &Coordinator{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.Shared.Monitoring),
	}
}

func (o *Coordinator) Run() error {
	go o.initializeCoordinator()
	go o.RunMonitoringServer()
	return nil
}

func (o *Coordinator) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for coordinator")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *Coordinator) Shutdown() {
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
	r.HandleFunc("/ws", server.WS)
	r.HandleFunc("/wso", server.WSO)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))
	r.PathPrefix("/").HandlerFunc(server.GetWeb)

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
	r.PathPrefix("/").HandlerFunc(handleRedirect)

	svmux := &http.ServeMux{}
	svmux.Handle("/", r)

	return makeServerFromMux(svmux)
}

// initializeCoordinator setup an coordinator server
func (o *Coordinator) initializeCoordinator() {
	// init games library
	lib := games.NewLibrary(games.Config{
		BasePath:  "assets/games",
		Supported: coordinator.SupportedRomExtensions,
		Ignored:   []string{"neogeo", "pgm"},
		Verbose:   true,
		WatchMode: o.cfg.LibraryMonitoring,
	})
	lib.Scan()

	server := NewServer(o.cfg, lib)

	var certManager *autocert.Manager
	var httpsSrv *http.Server

	log.Println("Initializing Coordinator Server")
	mode := o.cfg.Shared.Environment.Mode
	if mode.AnyOf(environment.Production, environment.Staging) {
		serverConfig := o.cfg.Shared.Server
		httpsSrv = makeHTTPServer(server)
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
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(o.cfg.PublicDomain),
				Cache:      autocert.DirCache("assets/cache"),
				Client:     &acme.Client{DirectoryURL: leurl},
			}

			httpsSrv.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
		}

		go func(chain string, key string) {
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			err := httpsSrv.ListenAndServeTLS(chain, key)
			if err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}(serverConfig.HttpsChain, serverConfig.HttpsKey)
	}

	var httpSrv *http.Server
	if mode.AnyOf(environment.Production, environment.Staging) {
		httpSrv = makeHTTPToHTTPSRedirectServer(server)
	} else {
		httpSrv = makeHTTPServer(server)
	}

	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = ":" + strconv.Itoa(o.cfg.Shared.Server.Port)
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}
