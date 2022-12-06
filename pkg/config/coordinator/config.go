package coordinator

import (
	"flag"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/games"
)

type Config struct {
	Coordinator Coordinator
	Emulator    emulator.Emulator
	Recording   shared.Recording
	Webrtc      webrtc.Webrtc
}

type Coordinator struct {
	RoundRobin bool
	Debug      bool
	DebugHost  string
	Library    games.Config
	Monitoring monitoring.Config
	Origin     struct {
		UserWs   string
		WorkerWs string
	}
	Server    shared.Server
	Analytics Analytics
}

// Analytics is optional Google Analytics
type Analytics struct {
	Inject bool
	Gtag   string
}

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	err := config.LoadConfig(&conf, configPath)
	if err != nil {
		panic(err)
	}
	conf.Webrtc.AddIceServersEnv()
	return
}

func (c *Config) ParseFlags() {
	c.Coordinator.Server.WithFlags()
	flag.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", c.Coordinator.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&configPath, "c-conf", configPath, "Set custom configuration file path")
	flag.Parse()
}
