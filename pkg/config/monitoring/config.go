package monitoring

type ServerMonitoringConfig struct {
	Port             int
	URLPrefix        string
	MetricEnabled    bool `json:"metric_enabled"`
	ProfilingEnabled bool `json:"profiling_enabled"`
}
