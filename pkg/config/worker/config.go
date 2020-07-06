package worker

import (
	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Port               int
	CoordinatorAddress string
	HttpPort           int

	// video
	Scale             int
	EnableAspectRatio bool
	Width             int
	Height            int
	Zone              string

	// WithoutGame to launch encoding with Game
	WithoutGame bool

	MonitoringConfig monitoring.ServerMonitoringConfig
}

func NewDefaultConfig() Config {
	return Config{
		Port:               8800,
		CoordinatorAddress: "localhost:8000",
		HttpPort:           9000,
		Scale:              1,
		EnableAspectRatio:  false,
		Width:              320,
		Height:             240,
		WithoutGame:        false,
		Zone:               "",
		MonitoringConfig: monitoring.ServerMonitoringConfig{
			Port:          6601,
			URLPrefix:     "/worker",
			MetricEnabled: true,
		},
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.IntVarP(&c.Port, "port", "", 8800, "Worker server port")
	fs.StringVarP(&c.CoordinatorAddress, "coordinatorhost", "", c.CoordinatorAddress, "Worker URL to connect")
	fs.IntVarP(&c.HttpPort, "httpPort", "", c.HttpPort, "Set external HTTP port")
	fs.StringVarP(&c.Zone, "zone", "z", c.Zone, "Zone of the worker")

	fs.IntVarP(&c.Scale, "scale", "s", c.Scale, "Set output viewport scale factor")
	fs.BoolVarP(&c.EnableAspectRatio, "ar", "", c.EnableAspectRatio, "Enable Aspect Ratio")
	fs.IntVarP(&c.Width, "width", "w", c.Width, "Set custom viewport width")
	fs.IntVarP(&c.Height, "height", "h", c.Height, "Set custom viewport height")
	fs.BoolVarP(&c.WithoutGame, "wogame", "", c.WithoutGame, "launch worker with game")

	fs.BoolVarP(&c.MonitoringConfig.MetricEnabled, "monitoring.metric", "m", c.MonitoringConfig.MetricEnabled, "Enable prometheus metric for server")
	fs.BoolVarP(&c.MonitoringConfig.ProfilingEnabled, "monitoring.pprof", "p", c.MonitoringConfig.ProfilingEnabled, "Enable golang pprof for server")
	fs.IntVarP(&c.MonitoringConfig.Port, "monitoring.port", "", c.MonitoringConfig.Port, "Monitoring server port")
	fs.StringVarP(&c.MonitoringConfig.URLPrefix, "monitoring.prefix", "", c.MonitoringConfig.URLPrefix, "Monitoring server url prefix")

	return c
}
