package overlord

import (
	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Port      int
	URLPrefix string

	MonitoringConfig monitoring.ServerMonitoringConfig
}

func NewDefaultConfig() Config {
	return Config{
		Port: 8800,

		MonitoringConfig: monitoring.ServerMonitoringConfig{
			Port:          6601,
			URLPrefix:     "/overlord",
			MetricEnabled: true,
		},
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.IntVarP(&c.Port, "port", "", 8800, "Overlord server port")

	fs.BoolVarP(&c.MonitoringConfig.MetricEnabled, "monitoring.metric", "m", c.MonitoringConfig.MetricEnabled, "Enable prometheus metric for server")
	fs.BoolVarP(&c.MonitoringConfig.ProfilingEnabled, "monitoring.pprof", "p", c.MonitoringConfig.ProfilingEnabled, "Enable golang pprof for server")
	fs.IntVarP(&c.MonitoringConfig.Port, "monitoring.port", "", c.MonitoringConfig.Port, "Monitoring server port")
	fs.StringVarP(&c.MonitoringConfig.URLPrefix, "monitoring.prefix", "", c.MonitoringConfig.URLPrefix, "Monitoring server url prefix")

	return c
}
