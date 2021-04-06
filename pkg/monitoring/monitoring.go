package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ServerMonitoring struct {
	cfg    monitoring.ServerMonitoringConfig
	tag    string
	server http.Server
}

func NewServerMonitoring(cfg monitoring.ServerMonitoringConfig, tag string) *ServerMonitoring {
	return &ServerMonitoring{cfg: validate(&cfg), tag: tag}
}

func (sm *ServerMonitoring) Init(conf interface{}) error {
	cfg := conf.(config.Config).Worker.Monitoring
	sm.cfg = validate(&cfg)
	return nil
}

func (sm *ServerMonitoring) Run() error {
	if sm.cfg.ProfilingEnabled || sm.cfg.MetricEnabled {
		monitoringServerMux := http.NewServeMux()

		srv := http.Server{
			Addr:    fmt.Sprintf(":%d", sm.cfg.Port),
			Handler: monitoringServerMux,
		}
		glog.Infof("[%v] Starting monitoring server at %v", sm.tag, srv.Addr)

		if sm.cfg.ProfilingEnabled {
			pprofPath := fmt.Sprintf("%s/debug/pprof", sm.cfg.URLPrefix)
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

		if sm.cfg.MetricEnabled {
			metricPath := fmt.Sprintf("%s/metrics", sm.cfg.URLPrefix)
			glog.Infof("[%v] Prometheus metric is enabled at %v", sm.tag, srv.Addr+metricPath)
			monitoringServerMux.Handle(metricPath, promhttp.Handler())
		}
		sm.server = srv
		return srv.ListenAndServe()
	}
	return nil
}

func (sm *ServerMonitoring) Shutdown(ctx context.Context) error {
	glog.Infof("[%v] Shutting down monitoring server", sm.tag)
	return sm.server.Shutdown(ctx)
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
