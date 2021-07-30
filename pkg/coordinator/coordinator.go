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

func (c *Coordinator) Run() error {
	go c.init()
	c.services.Start()
	return nil
}

func (c *Coordinator) init() {
	conf := c.conf.Coordinator.Server

	lib := getLibrary(&c.conf)
	lib.Scan()

	srv := NewServer(c.conf, lib)

	address := conf.Address
	if conf.Https {
		address = conf.Tls.Address
	}
	httpx.NewServer(
		address,
		func(_ *httpx.Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", index(c.conf))
			h.Handle("/static/", static("./web"))
			h.HandleFunc("/ws", srv.WS)
			h.HandleFunc("/wso", srv.WSO)
			return h
		},
		httpx.WithServerConfig(conf),
	).Start()
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
