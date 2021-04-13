package coordinator

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/server"
	"github.com/giongto35/cloud-game/v2/pkg/tls"
	"golang.org/x/crypto/acme/autocert"
)

type Coordinator struct {
	conf     coordinator.Config
	ctx      context.Context
	services server.Services
}

func New(ctx context.Context, conf coordinator.Config) *Coordinator {
	return &Coordinator{
		ctx:  ctx,
		conf: conf,
		services: []server.Server{
			monitoring.NewServerMonitoring(conf.Coordinator.Monitoring, "cord"),
		},
	}
}

func newServer(server *Server, addr string, redirectHTTPS bool) *http.Server {
	h := http.NewServeMux()

	base := index(server.cfg)
	if redirectHTTPS {
		base = redirect()
	}
	h.Handle("/", base)
	h.Handle("/static/", static("./web"))
	h.HandleFunc("/ws", server.WS)
	h.HandleFunc("/wso", server.WSO)

	// timeouts negate slow / frozen clients
	return &http.Server{
		Addr:         addr,
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

	if conf.Server.Https {
		// Letsencrypt or self
		var certManager *autocert.Manager
		if !conf.Server.Tls.IsSelfCert() {
			certManager = tls.NewTLSConfig(conf.Server.Tls.Domain).CertManager
		}

		go func() {
			serv := newServer(srv, conf.Server.Address, true)
			log.Printf("Starting HTTP->HTTPS server on %s", serv.Addr)
			if certManager != nil {
				serv.Handler = certManager.HTTPHandler(serv.Handler)
			}
			if err := serv.ListenAndServe(); err != nil {
				return
			}
		}()

		serv := newServer(srv, conf.Server.Address, false)
		if certManager != nil {
			serv.TLSConfig = certManager.TLSConfig()
		}
		log.Printf("Starting HTTPS server on %s", serv.Addr)
		if err := serv.ListenAndServeTLS(conf.Server.Tls.HttpsCert, conf.Server.Tls.HttpsKey); err != nil {
			log.Fatalf("error: %s", err)
		}
	} else {
		serv := newServer(srv, conf.Server.Address, false)
		log.Printf("Starting HTTP server on %s", serv.Addr)
		if err := serv.ListenAndServe(); err != nil {
			log.Fatalf("error: %s", err)
		}
	}
}

func (c *Coordinator) Shutdown() { c.services.Shutdown(c.ctx) }

// getLibrary initializes game library.
func getLibrary(conf *coordinator.Config) games.GameLibrary {
	libConf := conf.Coordinator.Library
	if len(libConf.Supported) == 0 {
		libConf.Supported = conf.Emulator.GetSupportedExtensions()
	}
	return games.NewLibrary(libConf)
}
