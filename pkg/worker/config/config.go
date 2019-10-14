package workerConfig

import (
	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Port            int
	OverlordAddress string

	// video
	Scale             int
	DisableCustomSize bool
	Width             int
	Height            int

	MonitoringConfig monitoring.ServerMonitoringConfig
}

func NewDefaultConfig() Config {
	return Config{
		Port:              8800,
		OverlordAddress:   "ws://localhost:8000/wso",
		Scale:             1,
		DisableCustomSize: false,
		Width:             320,
		Height:            240,
		MonitoringConfig: monitoring.ServerMonitoringConfig{
			Port:          6601,
			URLPrefix:     "/worker",
			MetricEnabled: true,
		},
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.IntVarP(&c.Port, "port", "", 8800, "OverWorker server port")
	fs.StringVarP(&c.OverlordAddress, "overlordhost", "", c.OverlordAddress, "OverWorker URL to connect")

	fs.IntVarP(&c.Scale, "scale", "s", c.Scale, "Set output viewport scale factor")
	fs.BoolVarP(&c.DisableCustomSize, "disable-custom-size", "", c.DisableCustomSize, "Disable custom size")
	fs.IntVarP(&c.Width, "width", "w", c.Width, "Set custom viewport width")
	fs.IntVarP(&c.Height, "height", "h", c.Height, "Set custom viewport height")

	fs.BoolVarP(&c.MonitoringConfig.MetricEnabled, "monitoring.metric", "m", c.MonitoringConfig.MetricEnabled, "Enable prometheus metric for server")
	fs.BoolVarP(&c.MonitoringConfig.ProfilingEnabled, "monitoring.pprof", "p", c.MonitoringConfig.ProfilingEnabled, "Enable golang pprof for server")
	fs.IntVarP(&c.MonitoringConfig.Port, "monitoring.port", "", c.MonitoringConfig.Port, "Monitoring server port")
	fs.StringVarP(&c.MonitoringConfig.URLPrefix, "monitoring.prefix", "", c.MonitoringConfig.URLPrefix, "Monitoring server url prefix")

	return c
}
