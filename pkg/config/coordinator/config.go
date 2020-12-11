package coordinator

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/spf13/pflag"
)

type Config struct {
	Shared shared.Config

	Coordinator struct {
		PublicDomain      string
		PingServer        string
		DebugHost         string
		LibraryMonitoring bool
		Monitoring        monitoring.ServerMonitoringConfig
	}
	Emulator emulator.Emulator
	Webrtc   struct {
		IceServers []webrtc.IceServer
	}
}

// allows custom config path
var configPath string

func NewDefaultConfig() *Config {
	var conf Config
	config.LoadConfig(&conf, configPath)
	log.Printf("%+v", conf)
	return &conf
}

func (c *Config) WithFlags(fs *pflag.FlagSet) *Config {
	c.Shared.AddFlags(fs)
	fs.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", 0, "Monitoring server port")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")
	return c
}
