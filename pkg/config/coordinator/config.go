package coordinator

import (
	"flag"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v3/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/config/shared"
	"github.com/giongto35/cloud-game/v3/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/games"
)

type Config struct {
	Coordinator Coordinator
	Emulator    emulator.Emulator
	Recording   shared.Recording
	Version     shared.Version
	Webrtc      webrtc.Webrtc
}

type Coordinator struct {
	Analytics  Analytics
	Debug      bool
	Library    games.Config
	Monitoring monitoring.Config
	Origin     struct {
		UserWs   string
		WorkerWs string
	}
	Selector string
	Server   shared.Server
}

// Analytics is optional Google Analytics
type Analytics struct {
	Inject bool
	Gtag   string
}

const SelectByPing = "ping"

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	err := config.LoadConfig(&conf, configPath)
	if err != nil {
		panic(err)
	}
	return
}

func (c *Config) ParseFlags() {
	c.Coordinator.Server.WithFlags()
	flag.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", c.Coordinator.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&configPath, "c-conf", configPath, "Set custom configuration file path")
	flag.Parse()
}
