package coordinator

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/covers"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/meta"
	"github.com/giongto35/cloud-game/v3/pkg/monitoring"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/store"
	"github.com/giongto35/cloud-game/v3/pkg/tournament"
)

type Coordinator struct {
	hub      *Hub
	tm       *tournament.Manager
	store    *store.Store
	cancel   context.CancelFunc
	services [2]interface {
		Run()
		Stop() error
	}
}

func New(conf config.CoordinatorConfig, log *logger.Logger) (*Coordinator, error) {
	dbPath := conf.Coordinator.Auth.DbPath
	if dbPath == "" {
		dbPath = "./data/belko.db"
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("database init fail (%s): %w", dbPath, err)
	}
	log.Info().Msgf("database: %s", dbPath)

	authSvc := auth.NewService(st, conf.Coordinator.Auth, log)
	if err := authSvc.EnsureAdmin(context.Background()); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("admin init fail: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go authSvc.RunJanitor(ctx)

	coordinator := &Coordinator{hub: NewHub(conf, authSvc, log), store: st, cancel: cancel}
	api := &API{auth: authSvc, hub: coordinator.hub, secure: conf.Coordinator.Auth.SecureCookie || conf.Coordinator.Server.Https, log: log}
	api.store = st
	if tc := conf.Coordinator.Tournaments; tc.Enabled {
		api.tm = tournament.NewManager(st, tournament.Config{
			MaxConcurrent: tc.MaxConcurrent,
			JoinTimeout:   time.Duration(tc.JoinTimeoutSec) * time.Second,
			ReplayDir:     tc.ReplayDir,
		}, log)
		coordinator.hub.tm = api.tm
		if err := api.tm.Start(coordinator.hub); err != nil {
			cancel()
			_ = st.Close()
			return nil, fmt.Errorf("tournaments init fail: %w", err)
		}
		coordinator.tm = api.tm
		log.Info().Msgf("tournaments: up to %d matches at once", api.tm.MaxConcurrent())
	}
	if conf.Coordinator.Covers.Enabled {
		api.covers = covers.New(st, conf.Coordinator.Covers, log)
		coordinator.hub.covers = api.covers
		log.Info().Msgf("covers: %s -> %s", conf.Coordinator.Covers.Url, conf.Coordinator.Covers.Dir)
	}

	if conf.Coordinator.Meta.Enabled {
		api.meta = meta.New(st, conf.Coordinator.Meta, log)
		coordinator.hub.meta = api.meta
		log.Info().Msgf("catalog: %s -> %s", conf.Coordinator.Meta.Url, conf.Coordinator.Meta.Dir)
	}

	h, err := NewHTTPServer(conf, log, api, func(mux *httpx.Mux) *httpx.Mux {
		mux.HandleFunc("/ws", coordinator.hub.handleUserConnection())
		mux.HandleFunc("/wso", coordinator.hub.handleWorkerConnection())
		api.Routes(mux)
		return mux
	})
	if err != nil {
		cancel()
		_ = st.Close()
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
	if c.tm != nil {
		c.tm.Stop()
	}
	c.hub.Stop()
	c.cancel()
	err = errors.Join(err, c.store.Close())
	return err
}

func NewHTTPServer(conf config.CoordinatorConfig, log *logger.Logger, api *API, fnMux func(*httpx.Mux) *httpx.Mux) (*httpx.Server, error) {
	return httpx.NewServer(
		conf.Coordinator.Server.GetAddr(),
		func(s *httpx.Server) httpx.Handler { return fnMux(s.Mux().Handle("/", index(conf, api, log))) },
		httpx.WithServerConfig(conf.Coordinator.Server),
		httpx.WithLogger(log),
	)
}

func securityHeaders(w http.ResponseWriter, conf config.CoordinatorConfig) {
	hd := w.Header()
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Referrer-Policy", "same-origin")
	if conf.Coordinator.Server.FrameOptions != "" {
		hd.Set("X-Frame-Options", conf.Coordinator.Server.FrameOptions)
	}
}

func index(conf config.CoordinatorConfig, api *API, log *logger.Logger) httpx.Handler {
	const indexHTML = "./web/index.html"
	const authHTML = "./web/auth.html"

	tplData := struct {
		Analytics config.Analytics
		Recording config.Recording
	}{conf.Coordinator.Analytics, conf.Recording}

	parse := func() (*template.Template, *template.Template) {
		return template.Must(template.ParseFiles(indexHTML)), template.Must(template.ParseFiles(authHTML))
	}
	indexTpl, authTpl := parse()
	if conf.Coordinator.Debug {
		log.Info().Msgf("Using auto-reloading index.html")
	}

	render := func(tpl *template.Template, w httpx.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := tpl.Execute(w, tplData); err != nil {
			log.Error().Err(err).Msg("error with the page template file")
		}
	}

	files := httpx.FileServer("./web")

	return httpx.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		securityHeaders(w, conf)
		it, at := indexTpl, authTpl
		if conf.Coordinator.Debug {
			it, at = parse()
		}
		path := r.URL.Path
		switch {
		case path == "/" || strings.HasSuffix(path, "/index.html"):
			if target := api.gate(r); target != "" {
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, target, http.StatusSeeOther)
				return
			}
			render(it, w)
		case path == "/entrar" || path == "/auth.html":
			render(at, w)
		default:
			if conf.Coordinator.Server.CacheControl != "" {
				w.Header().Add("Cache-Control", conf.Coordinator.Server.CacheControl)
			}
			files.ServeHTTP(w, r)
		}
	})
}
