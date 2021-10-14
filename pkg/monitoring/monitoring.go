package monitoring

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const debugEndpoint = "/debug/pprof"
const metricsEndpoint = "/metrics"

type Monitoring struct {
	service.RunnableService

	conf   monitoring.Config
	server *httpx.Server
	log    *logger.Logger
}

// New creates new monitoring service.
// The tag param specifies owner label for logs.
func New(conf monitoring.Config, baseAddr string, log *logger.Logger) *Monitoring {
	serv, err := httpx.NewServer(
		net.JoinHostPort(baseAddr, strconv.Itoa(conf.Port)),
		func(*httpx.Server) http.Handler {
			h := http.NewServeMux()
			if conf.ProfilingEnabled {
				prefix := conf.URLPrefix + debugEndpoint
				h.HandleFunc(prefix+"/", pprof.Index)
				h.HandleFunc(prefix+"/cmdline", pprof.Cmdline)
				h.HandleFunc(prefix+"/profile", pprof.Profile)
				h.HandleFunc(prefix+"/symbol", pprof.Symbol)
				h.HandleFunc(prefix+"/trace", pprof.Trace)
				h.Handle(prefix+"/allocs", pprof.Handler("allocs"))
				h.Handle(prefix+"/block", pprof.Handler("block"))
				h.Handle(prefix+"/goroutine", pprof.Handler("goroutine"))
				h.Handle(prefix+"/heap", pprof.Handler("heap"))
				h.Handle(prefix+"/mutex", pprof.Handler("mutex"))
				h.Handle(prefix+"/threadcreate", pprof.Handler("threadcreate"))
			}
			if conf.MetricEnabled {
				h.Handle(conf.URLPrefix+metricsEndpoint, promhttp.Handler())
			}
			return h
		},
		httpx.WithPortRoll(true),
		httpx.WithLogger(log),
	)
	if err != nil {
		log.Error().Err(err).Msg("couldn't start monitoring server")
	}
	return &Monitoring{conf: conf, server: serv, log: log}
}

func (m *Monitoring) Run() {
	m.printInfo()
	m.server.Run()
}

func (m *Monitoring) Shutdown(ctx context.Context) error {
	m.log.Info().Msg("Shutting down monitoring server")
	return m.server.Shutdown(ctx)
}

func (m *Monitoring) String() string {
	return fmt.Sprintf("monitoring::%s:%d", m.conf.URLPrefix, m.conf.Port)
}

func (m *Monitoring) GetMetricsPublicAddress() string {
	return m.server.GetProtocol() + "://" + m.server.Addr + m.conf.URLPrefix + metricsEndpoint
}

func (m *Monitoring) GetProfilingAddress() string {
	return m.server.GetProtocol() + "://" + m.server.Addr + m.conf.URLPrefix + debugEndpoint
}

func (m *Monitoring) printInfo() {
	message := m.log.Info()
	if m.conf.ProfilingEnabled {
		message = message.Str("profiler", m.GetProfilingAddress())
	}
	if m.conf.MetricEnabled {
		message = message.Str("prometheus", m.GetMetricsPublicAddress())
	}

	message.Msg("Monitoring")
}
