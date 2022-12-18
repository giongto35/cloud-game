package monitoring

import (
	"context"
	"fmt"
	"net"
	"net/http/pprof"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
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
		func(s *httpx.Server) httpx.Handler {
			h := s.Mux()
			if conf.ProfilingEnabled {
				h.Prefix(conf.URLPrefix + debugEndpoint)
				h.HandleFunc("/", pprof.Index).
					HandleFunc("/cmdline", pprof.Cmdline).
					HandleFunc("/profile", pprof.Profile).
					HandleFunc("/symbol", pprof.Symbol).
					HandleFunc("/trace", pprof.Trace).
					Handle("/allocs", pprof.Handler("allocs")).
					Handle("/block", pprof.Handler("block")).
					Handle("/goroutine", pprof.Handler("goroutine")).
					Handle("/heap", pprof.Handler("heap")).
					Handle("/mutex", pprof.Handler("mutex")).
					Handle("/threadcreate", pprof.Handler("threadcreate"))
			}
			if conf.MetricEnabled {
				h.Prefix(conf.URLPrefix)
				h.HandleFunc(metricsEndpoint, func(w httpx.ResponseWriter, _ *httpx.Request) {
					metrics.WritePrometheus(w, true)
				})
			}
			h.Prefix("")
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
