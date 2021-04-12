package coordinator

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/golang/glog"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type Coordinator struct {
	cfg              coordinator.Config
	ctx              context.Context
	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg coordinator.Config) *Coordinator {
	return &Coordinator{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.Coordinator.Monitoring, "cord"),
	}
}

func (c *Coordinator) Run() error {
	go c.init()
	go c.RunMonitoringServer()
	return nil
}

func (c *Coordinator) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for coordinator")
	err := c.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (c *Coordinator) Shutdown() {
	if err := c.monitoringServer.Shutdown(c.ctx); err != nil {
		glog.Errorln("Failed to shutdown monitoring server")
	}
}

func newServer(server *Server, redirectHTTPS bool) *http.Server {
	h := http.NewServeMux()

	base := index
	if redirectHTTPS {
		base = redirect
	}
	h.HandleFunc("/", base)
	h.Handle("/static/", static("./web"))
	h.HandleFunc("/ws", server.WS)
	h.HandleFunc("/wso", server.WSO)

	// timeouts negate slow / frozen clients
	return &http.Server{
		Handler:      h,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

func (c *Coordinator) init() {
	conf := c.cfg.Coordinator
	// init games library
	if len(conf.Library.Supported) == 0 {
		conf.Library.Supported = c.cfg.Emulator.GetSupportedExtensions()
	}
	lib := games.NewLibrary(conf.Library)
	lib.Scan()

	server := NewServer(c.cfg, lib)

	var certManager *autocert.Manager
	var httpsSrv *http.Server

	log.Println("Initializing Coordinator Server")
	mode := c.cfg.Environment.Get()
	if mode.AnyOf(environment.Production, environment.Staging) {
		httpsSrv = newServer(server, false)
		httpsSrv.Addr = strconv.Itoa(conf.Server.HttpsPort)

		if conf.Server.HttpsChain == "" || conf.Server.HttpsKey == "" {
			conf.Server.HttpsChain = ""
			conf.Server.HttpsKey = ""

			letsEncryptURL := acme.LetsEncryptURL
			if mode == environment.Staging {
				letsEncryptURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
			}
			certManager = &autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(conf.PublicDomain),
				Cache:      autocert.DirCache("assets/cache"),
				Client:     &acme.Client{DirectoryURL: letsEncryptURL},
			}
			httpsSrv.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
		}

		go func() {
			log.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			if err := httpsSrv.ListenAndServeTLS(conf.Server.HttpsChain, conf.Server.HttpsKey); err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	}

	httpSrv := newServer(server, mode.AnyOf(environment.Production, environment.Staging))
	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}
	httpSrv.Addr = ":" + strconv.Itoa(conf.Server.Port)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}
