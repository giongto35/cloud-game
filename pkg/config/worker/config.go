package worker

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Shared shared.Config

	Encoder struct {
		WithoutGame bool
	}
	Emulator emulator.Emulator
	Worker   struct {
		Monitoring monitoring.ServerMonitoringConfig
		Network    struct {
			CoordinatorAddress string
			Zone               string
		}
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
	if err := fs.Set("port", "9000"); err != nil {
		log.Printf("error: couldn't override default port value, %v", err)
	}
	fs.IntVar(&c.Worker.Monitoring.Port, "monitoring.port", 0, "Monitoring server port")
	fs.StringVar(&c.Worker.Network.CoordinatorAddress, "coordinatorhost", "", "Worker URL to connect")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")
	return c
}
