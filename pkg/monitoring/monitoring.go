package monitoring

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const debugEndpoint = "/debug/pprof"
const metricsEndpoint = "/metrics"

type Monitoring struct {
	service.RunnableService

	conf   monitoring.Config
	tag    string
	server *httpx.Server
}

// New creates new monitoring service.
// The tag param specifies owner label for logs.
func New(conf monitoring.Config, baseAddr string, tag string) *Monitoring {
	serv, err := httpx.NewServer(
		net.JoinHostPort(baseAddr, strconv.Itoa(conf.Port)),
		func(serv *httpx.Server) http.Handler {
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
		httpx.WithPortRoll(true))
	if err != nil {
		panic("couldn't start monitoring server: " + err.Error())
	}
	return &Monitoring{conf: conf, tag: tag, server: serv}
}

func (m *Monitoring) Run() {
	m.printInfo()
	m.server.Run()
}

func (m *Monitoring) Shutdown(ctx context.Context) error {
	log.Printf("[%v] Shutting down monitoring server", m.tag)
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
	length, pad := 42, 20
	var table, records strings.Builder
	table.Grow(length * 4)
	records.Grow(length * 2)

	if m.conf.ProfilingEnabled {
		addr := m.GetProfilingAddress()
		length = int(math.Max(float64(length), float64(len(addr)+pad)))
		records.WriteString("    Profiling   " + addr + "\n")
	}
	if m.conf.MetricEnabled {
		addr := m.GetMetricsPublicAddress()
		length = int(math.Max(float64(length), float64(len(addr)+pad)))
		records.WriteString("    Prometheus  " + addr + "\n")
	}

	title := "Monitoring"
	center := strconv.Itoa(length / 2)
	rPad := strconv.Itoa((length / 2) - (len(title) / 2) - 1)
	table.WriteString(fmt.Sprintf("[%s]\n", m.tag))
	table.WriteString(strings.Repeat("-", length) + "\n")
	table.WriteString(fmt.Sprintf("---%"+center+"s%"+rPad+"s---\n", title, ""))
	table.WriteString(strings.Repeat("-", length) + "\n")
	table.WriteString(records.String())
	table.WriteString(strings.Repeat("-", length))
	log.Printf(table.String())
}
