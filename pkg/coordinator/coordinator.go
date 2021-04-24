package coordinator

import (
	"context"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/server"
)

type Coordinator struct {
	conf     coordinator.Config
	ctx      context.Context
	services *server.Services
}

func New(ctx context.Context, conf coordinator.Config) *Coordinator {
	services := server.Services{}
	return &Coordinator{
		ctx:  ctx,
		conf: conf,
		services: services.AddIf(
			conf.Coordinator.Monitoring.IsEnabled(), monitoring.New(conf.Coordinator.Monitoring, "cord"),
		),
	}
}

func (c *Coordinator) Run() {
	conf := c.conf.Coordinator.Server

	lib := c.getLibrary()
	lib.Scan()

	hub := NewHub(c.conf, lib)

	address := conf.Address
	if conf.Https {
		address = conf.Tls.Address
	}
	go httpx.NewServer(
		address,
		func(_ *httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(c.conf))
			h.Handle("/static/", static("./web"))
			h.HandleFunc("/ws", hub.handleNewWebsocketUserConnection)
			h.HandleFunc("/wso", hub.handleNewWebsocketWorkerConnection)
			return h
		},
		httpx.WithServerConfig(conf),
	).Start()

	c.services.Start()
}

func (c *Coordinator) Shutdown() { c.services.Shutdown(c.ctx) }

// !to rewrite
// getLibrary scans files with explicit extensions or
// the extensions specified in the emulator config.
func (c *Coordinator) getLibrary() games.GameLibrary {
	libConf := c.conf.Coordinator.Library
	if len(libConf.Supported) == 0 {
		libConf.Supported = c.conf.Emulator.GetSupportedExtensions()
	}
	return games.NewLibrary(libConf)
}
