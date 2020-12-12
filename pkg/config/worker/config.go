package worker

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Encoder     encoder.Encoder
	Emulator    emulator.Emulator
	Environment shared.Environment
	Worker      struct {
		Monitoring monitoring.ServerMonitoringConfig
		Network    struct {
			CoordinatorAddress string
			Zone               string
		}
		Server shared.Server
	}
	Webrtc webrtcConfig.Webrtc
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
	c.Worker.Server.WithFlags(fs)
	fs.IntVar(&c.Worker.Monitoring.Port, "monitoring.port", 0, "Monitoring server port")
	fs.StringVar(&c.Worker.Network.CoordinatorAddress, "coordinatorhost", "", "Worker URL to connect")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")
	return c
}
