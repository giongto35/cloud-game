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

		monitoringServer: monitoring.NewServerMonitoring(cfg.Coordinator.Monitoring, "cord"),
	}
}

func (c *Coordinator) Run() error {
	go c.initializeCoordinator()
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

// initializeCoordinator setup an coordinator server
func (c *Coordinator) initializeCoordinator() {
	// init games library
	libraryConf := c.cfg.Coordinator.Library
	if len(libraryConf.Supported) == 0 {
		libraryConf.Supported = c.cfg.Emulator.GetSupportedExtensions()
	}
	lib := games.NewLibrary(libraryConf)
	lib.Scan()

	server := NewServer(c.cfg, lib)

	var certManager *autocert.Manager
	var httpsSrv *http.Server

	log.Println("Initializing Coordinator Server")
	mode := c.cfg.Environment.Get()
	if mode.AnyOf(environment.Production, environment.Staging) {
		serverConfig := c.cfg.Coordinator.Server
		httpsSrv = newServer(server, false)
		httpsSrv.Addr = strconv.Itoa(serverConfig.HttpsPort)

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
				HostPolicy: autocert.HostWhitelist(c.cfg.Coordinator.PublicDomain),
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
		httpSrv = newServer(server, true)
	} else {
		httpSrv = newServer(server, false)
	}

	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = ":" + strconv.Itoa(c.cfg.Coordinator.Server.Port)
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}
