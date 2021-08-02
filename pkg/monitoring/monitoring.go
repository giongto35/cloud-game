package monitoring

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Monitoring struct {
	service.RunnableService

	conf   monitoring.Config
	tag    string
	server *httpx.Server
}

// New creates new monitoring service.
// The tag param specifies owner label for logs.
func New(conf monitoring.Config, tag string) *Monitoring {
	serv, _ := httpx.NewServer(
		fmt.Sprintf(":%d", conf.Port),
		func(serv *httpx.Server) http.Handler {
			h := http.NewServeMux()

			if conf.ProfilingEnabled {
				prefix := fmt.Sprintf("%s/debug/pprof", conf.URLPrefix)
				log.Printf("[%v] Profiling is enabled at %v", tag, serv.Addr+prefix)
				h.HandleFunc(prefix+"/", pprof.Index)
				h.HandleFunc(prefix+"/cmdline", pprof.Cmdline)
				h.HandleFunc(prefix+"/profile", pprof.Profile)
				h.HandleFunc(prefix+"/symbol", pprof.Symbol)
				h.HandleFunc(prefix+"/trace", pprof.Trace)
				// pprof handler for custom pprof path needs to be explicitly specified,
				// according to: https://github.com/gin-contrib/pprof/issues/8.
				// Don't know why this is not fired as ticket
				// https://golang.org/src/net/http/pprof/pprof.go?s=7411:7461#L305 only render index page
				h.Handle(prefix+"/allocs", pprof.Handler("allocs"))
				h.Handle(prefix+"/block", pprof.Handler("block"))
				h.Handle(prefix+"/goroutine", pprof.Handler("goroutine"))
				h.Handle(prefix+"/heap", pprof.Handler("heap"))
				h.Handle(prefix+"/mutex", pprof.Handler("mutex"))
				h.Handle(prefix+"/threadcreate", pprof.Handler("threadcreate"))
			}

			if conf.MetricEnabled {
				metricPath := fmt.Sprintf("%s/metrics", conf.URLPrefix)
				log.Printf("[%v] Prometheus metric is enabled at %v", tag, serv.Addr+metricPath)
				h.Handle(metricPath, promhttp.Handler())
			}

			return h
		},
	)
	return &Monitoring{conf: conf, tag: tag, server: serv}
}

func (m *Monitoring) Run() {
	log.Printf("[%v] Starting monitoring server at %v", m.tag, m.server.Addr)
	m.server.Run()
}

func (m *Monitoring) Shutdown(ctx context.Context) error {
	log.Printf("[%v] Shutting down monitoring server", m.tag)
	return m.server.Shutdown(ctx)
}

func (m *Monitoring) String() string {
	return fmt.Sprintf("monitoring::%s:%d", m.conf.URLPrefix, m.conf.Port)
}
