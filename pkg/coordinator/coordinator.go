package coordinator

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/server"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type Coordinator struct {
	conf     coordinator.Config
	ctx      *context.Context
	services server.Services
}

func New(ctx context.Context, conf coordinator.Config) *Coordinator {
	return &Coordinator{
		ctx:  &ctx,
		conf: conf,
		services: []server.Server{
			monitoring.NewServerMonitoring(conf.Coordinator.Monitoring, "cord"),
		},
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

func (c *Coordinator) Run() error {
	go c.init()
	c.services.Start()
	return nil
}

func (c *Coordinator) init() {
	conf := c.conf.Coordinator

	lib := getLibrary(&c.conf)
	lib.Scan()

	srv := NewServer(c.conf, lib)

	var certManager *autocert.Manager
	var httpsSrv *http.Server

	mode := c.conf.Environment.Get()
	if mode.AnyOf(environment.Production, environment.Staging) {
		httpsSrv = newServer(srv, false)
		httpsSrv.Addr = conf.Server.HttpsAddress

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

	httpSrv := newServer(srv, mode.AnyOf(environment.Production, environment.Staging))
	if certManager != nil {
		httpSrv.Handler = certManager.HTTPHandler(httpSrv.Handler)
	}
	httpSrv.Addr = conf.Server.Address
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

func (c *Coordinator) Shutdown() { c.services.Shutdown(c.ctx) }

// getLibrary initializes games library.
func getLibrary(conf *coordinator.Config) games.GameLibrary {
	libConf := conf.Coordinator.Library
	if len(libConf.Supported) == 0 {
		libConf.Supported = conf.Emulator.GetSupportedExtensions()
	}
	return games.NewLibrary(libConf)
}
