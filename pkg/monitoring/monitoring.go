package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ServerMonitoring struct {
	conf   monitoring.ServerMonitoringConfig
	tag    string
	server *http.Server
}

// NewServerMonitoring creates new monitoring service.
// The tag param specifies owner label for logs.
func NewServerMonitoring(conf monitoring.ServerMonitoringConfig, tag string) *ServerMonitoring {
	return &ServerMonitoring{conf: validate(&conf), tag: tag}
}

func (sm *ServerMonitoring) Run() error {
	if sm.conf.ProfilingEnabled || sm.conf.MetricEnabled {
		monitoringServerMux := http.NewServeMux()

		srv := http.Server{
			Addr:    fmt.Sprintf(":%d", sm.conf.Port),
			Handler: monitoringServerMux,
		}
		sm.server = &srv
		glog.Infof("[%v] Starting monitoring server at %v", sm.tag, srv.Addr)

		if sm.conf.ProfilingEnabled {
			pprofPath := fmt.Sprintf("%s/debug/pprof", sm.conf.URLPrefix)
			glog.Infof("[%v] Profiling is enabled at %v", sm.tag, srv.Addr+pprofPath)
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

		if sm.conf.MetricEnabled {
			metricPath := fmt.Sprintf("%s/metrics", sm.conf.URLPrefix)
			glog.Infof("[%v] Prometheus metric is enabled at %v", sm.tag, srv.Addr+metricPath)
			monitoringServerMux.Handle(metricPath, promhttp.Handler())
		}

		err := srv.ListenAndServe()
		if err == http.ErrServerClosed {
			glog.Infof("[%v] The main HTTP server has been closed", sm.tag)
			return nil
		}
		return err
	}
	return nil
}

func (sm *ServerMonitoring) Shutdown(ctx *context.Context) error {
	if sm.server == nil {
		return nil
	}

	glog.Infof("[%v] Shutting down monitoring server", sm.tag)
	return sm.server.Shutdown(*ctx)
}

func (sm *ServerMonitoring) String() string {
	return fmt.Sprintf("monitoring::%s:%d", sm.conf.URLPrefix, sm.conf.Port)
}

func validate(conf *monitoring.ServerMonitoringConfig) monitoring.ServerMonitoringConfig {
	if conf.Port == 0 {
		conf.Port = 6365
	}

	if len(conf.URLPrefix) > 0 {
		conf.URLPrefix = strings.TrimSpace(conf.URLPrefix)
		if !strings.HasPrefix(conf.URLPrefix, "/") {
			conf.URLPrefix = "/" + conf.URLPrefix
		}

		if strings.HasSuffix(conf.URLPrefix, "/") {
			conf.URLPrefix = strings.TrimSuffix(conf.URLPrefix, "/")
		}
	}
	return *conf
}
