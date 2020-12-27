package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ServerMonitoringConfig struct {
	Port             int
	URLPrefix        string
	MetricEnabled    bool `json:"metric_enabled"`
	ProfilingEnabled bool `json:"profiling_enabled"`
}

type ServerMonitoring struct {
	cfg ServerMonitoringConfig

	server http.Server
}

func NewServerMonitoring(cfg ServerMonitoringConfig) *ServerMonitoring {
	if cfg.Port == 0 {
		cfg.Port = 6365
	}

	if len(cfg.URLPrefix) > 0 {
		cfg.URLPrefix = strings.TrimSpace(cfg.URLPrefix)
		if !strings.HasPrefix(cfg.URLPrefix, "/") {
			cfg.URLPrefix = "/" + cfg.URLPrefix
		}

		if strings.HasSuffix(cfg.URLPrefix, "/") {
			cfg.URLPrefix = strings.TrimSuffix(cfg.URLPrefix, "/")
		}
	}
	return &ServerMonitoring{cfg: cfg}
}

func (sm *ServerMonitoring) Run() error {
	if sm.cfg.ProfilingEnabled || sm.cfg.MetricEnabled {
		monitoringServerMux := http.NewServeMux()

		srv := http.Server{
			Addr:    fmt.Sprintf(":%d", sm.cfg.Port),
			Handler: monitoringServerMux,
		}
		glog.Infoln("Starting monitoring server at", srv.Addr)

		if sm.cfg.ProfilingEnabled {
			pprofPath := fmt.Sprintf("%s/debug/pprof", sm.cfg.URLPrefix)
			glog.Infoln("Profiling is enabled at", srv.Addr+pprofPath)
			monitoringServerMux.Handle(pprofPath+"/", http.HandlerFunc(pprof.Index))
			monitoringServerMux.Handle(pprofPath+"/cmdline", http.HandlerFunc(pprof.Cmdline))
			monitoringServerMux.Handle(pprofPath+"/profile", http.HandlerFunc(pprof.Profile))
			monitoringServerMux.Handle(pprofPath+"/symbol", http.HandlerFunc(pprof.Symbol))
			monitoringServerMux.Handle(pprofPath+"/trace", http.HandlerFunc(pprof.Trace))
			// pprof handler for custom pprof path needs to be explicitly specified, according to: https://github.com/gin-contrib/pprof/issues/8 . Don't know why this is not fired as ticket
			// https://golang.org/src/net/http/pprof/pprof.go?s=7411:7461#L305 only render index page
			monitoringServerMux.Handle(pprofPath+"/allocs", pprof.Handler("allocs"))
			monitoringServerMux.Handle(pprofPath+"/block", pprof.Handler("block"))
			monitoringServerMux.Handle(pprofPath+"/goroutine", pprof.Handler("goroutine"))
			monitoringServerMux.Handle(pprofPath+"/heap", pprof.Handler("heap"))
			monitoringServerMux.Handle(pprofPath+"/mutex", pprof.Handler("mutex"))
			monitoringServerMux.Handle(pprofPath+"/threadcreate", pprof.Handler("threadcreate"))
		}

		if sm.cfg.MetricEnabled {
			metricPath := fmt.Sprintf("%s/metrics", sm.cfg.URLPrefix)
			glog.Infoln("Prometheus metric is enabled at", srv.Addr+metricPath)
			monitoringServerMux.Handle(metricPath, promhttp.Handler())
		}
		sm.server = srv
		return srv.ListenAndServe()
	}
	glog.Infoln("Monitoring server is disabled via config")
	return nil
}

func (sm *ServerMonitoring) Shutdown(ctx context.Context) error {
	glog.Infoln("Shutting down monitoring server")
	return sm.server.Shutdown(ctx)
}
