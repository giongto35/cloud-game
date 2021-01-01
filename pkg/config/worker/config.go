package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	flag "github.com/spf13/pflag"
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
	Loaded bool
}

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	if err := config.LoadConfig(&conf, configPath); err == nil {
		conf.Loaded = false
	}
	return
}

// ParseFlags updates config values from passed runtime flags.
// Define own flags with default value set to the current config param.
// Don't forget to call flag.Parse().
func (c *Config) ParseFlags() {
	c.Environment.WithFlags()
	c.Worker.Server.WithFlags()
	flag.IntVar(&c.Worker.Monitoring.Port, "monitoring.port", c.Worker.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&c.Worker.Network.CoordinatorAddress, "coordinatorhost", c.Worker.Network.CoordinatorAddress, "Worker URL to connect")
	flag.StringVar(&c.Worker.Network.Zone, "zone", c.Worker.Network.Zone, "Worker network zone (us, eu, etc.)")
	flag.StringVarP(&configPath, "conf", "c", configPath, "Set custom configuration file path")
	flag.Parse()
}
