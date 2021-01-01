package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Coordinator struct {
		PublicDomain string
		PingServer   string
		DebugHost    string
		Library      games.Config
		Monitoring   monitoring.ServerMonitoringConfig
		Server       shared.Server
	}
	Emulator    emulator.Emulator
	Environment shared.Environment
	Webrtc      struct {
		IceServers []webrtcConfig.IceServer
	}
}

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
	c.Environment.WithFlags()
	c.Coordinator.Server.WithFlags()
	flag.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", c.Coordinator.Monitoring.Port, "Monitoring server port")
	flag.StringVarP(&configPath, "conf", "c", configPath, "Set custom configuration file path")
	flag.Parse()
}
