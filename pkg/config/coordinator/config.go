package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
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

func NewConfig() *Config {
	var conf Config
	config.LoadConfig(&conf, configPath)
	log.Printf("%+v", conf)
	return &conf
}

func (c *Config) WithFlags(fs *pflag.FlagSet) *Config {
	c.Environment.WithFlags(fs)
	c.Coordinator.Server.WithFlags(fs)
	fs.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", 0, "Monitoring server port")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")
	return c
}
