package coordinator

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
)

type Coordinator struct {
	hub      *Hub
	services [2]interface {
		Run()
		Stop() error
	}
}

func New(conf config.CoordinatorConfig, log *logger.Logger) (*Coordinator, error) {
	coordinator := &Coordinator{}
	lib := games.NewLib(conf.Coordinator.Library, conf.Emulator, log)
	lib.Scan()
	coordinator.hub = NewHub(conf, lib, log)
	h, err := NewHTTPServer(conf, log, func(mux *httpx.Mux) *httpx.Mux {
		mux.HandleFunc("/ws", coordinator.hub.handleUserConnection())
		mux.HandleFunc("/wso", coordinator.hub.handleWorkerConnection())
		return mux
	})
	if err != nil {
		return nil, fmt.Errorf("http init fail: %w", err)
	}
	coordinator.services[0] = h
	if conf.Coordinator.Monitoring.IsEnabled() {
		coordinator.services[1] = monitoring.New(conf.Coordinator.Monitoring, h.GetHost(), log)
	}
	return coordinator, nil
}

func (c *Coordinator) Start() {
	for _, s := range c.services {
		if s != nil {
			s.Run()
		}
	}
}

func (c *Coordinator) Stop() error {
	var err error
	for _, s := range c.services {
		if s != nil {
			err0 := s.Stop()
			err = errors.Join(err, err0)
		}
	}
	return err
}

func NewHTTPServer(conf config.CoordinatorConfig, log *logger.Logger, fnMux func(*httpx.Mux) *httpx.Mux) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(s *httpx.Server) httpx.Handler { return fnMux(s.Mux().Handle("/", index(conf, log))) },
		httpx.WithServerConfig(conf.Coordinator.Server),
		httpx.WithLogger(log),
	)
}

func index(conf config.CoordinatorConfig, log *logger.Logger) httpx.Handler {
	const indexHTML = "./web/index.html"

	indexTpl := template.Must(template.ParseFiles(indexHTML))

	// render index page with some tpl values
	tplData := struct {
		Analytics config.Analytics
		Recording config.Recording
	}{conf.Coordinator.Analytics, conf.Recording}

	handler := func(tpl *template.Template, w httpx.ResponseWriter, r *httpx.Request) {
		if err := tpl.Execute(w, tplData); err != nil {
			log.Fatal().Err(err).Msg("error with the analytics template file")
		}
	}

	h := httpx.FileServer("./web")

	if conf.Coordinator.Debug {
		log.Info().Msgf("Using auto-reloading index.html")
		return httpx.HandlerFunc(func(w httpx.ResponseWriter, r *httpx.Request) {
			if conf.Coordinator.Server.CacheControl != "" {
				w.Header().Add("Cache-Control", conf.Coordinator.Server.CacheControl)
			}
			if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/index.html") {
				tpl := template.Must(template.ParseFiles(indexHTML))
				handler(tpl, w, r)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	return httpx.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if conf.Coordinator.Server.CacheControl != "" {
			w.Header().Add("Cache-Control", conf.Coordinator.Server.CacheControl)
		}
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/index.html") {
			handler(indexTpl, w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
