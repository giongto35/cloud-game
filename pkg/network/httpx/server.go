package httpx

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	http.Server

	autoCert *autocert.Manager
	opts     Options

	listener *Listener
	redirect *Server
	log      *logger.Logger
}

type (
	Mux struct {
		*http.ServeMux
		prefix string
	}
	Handler        = http.Handler
	HandlerFunc    = http.HandlerFunc
	ResponseWriter = http.ResponseWriter
	Request        = http.Request
)

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux(prefix string) *Mux {
	return &Mux{ServeMux: http.NewServeMux(), prefix: prefix}
}

func (m *Mux) Prefix(v string) { m.prefix = v }

func (m *Mux) HandleW(pattern string, h func(http.ResponseWriter)) *Mux {
	m.ServeMux.HandleFunc(m.prefix+pattern, func(w http.ResponseWriter, _ *http.Request) { h(w) })
	return m
}

func (m *Mux) Handle(pattern string, handler Handler) *Mux {
	m.ServeMux.Handle(m.prefix+pattern, handler)
	return m
}

func (m *Mux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) *Mux {
	m.ServeMux.HandleFunc(m.prefix+pattern, handler)
	return m
}

func (m *Mux) ServeHTTP(w ResponseWriter, r *Request) { m.ServeMux.ServeHTTP(w, r) }

func NewServer(address string, handler func(*Server) Handler, options ...Option) (*Server, error) {
	opts := &Options{
		Https:         false,
		HttpsRedirect: true,
		IdleTimeout:   120 * time.Second,
		ReadTimeout:   500 * time.Second,
		WriteTimeout:  500 * time.Second,
	}
	opts.override(options...)

	if opts.Logger == nil {
		opts.Logger = logger.Default()
	}

	server := &Server{
		Server: http.Server{
			Addr:         address,
			IdleTimeout:  opts.IdleTimeout,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
		},
		opts: *opts,
		log:  opts.Logger,
	}
	// (╯°□°)╯︵ ┻━┻
	server.Handler = handler(server)

	if opts.Https && opts.IsAutoHttpsCert() {
		server.autoCert = NewTLSConfig(withZonePrefix(opts.HttpsDomain, opts.Zone)).CertManager
		server.TLSConfig = server.autoCert.TLSConfig()
	}

	addr := server.Addr
	if server.Addr == "" {
		addr = ":http"
		if opts.Https {
			addr = ":https"
		}
		opts.Logger.Warn().Msgf("Empty server address has been changed to %v", addr)
	}
	listener, err := NewListener(addr, server.opts.PortRoll)
	if err != nil {
		return nil, err
	}
	server.listener = listener

	addr = buildAddress(server.Addr, opts.Zone, *listener)
	opts.Logger.Info().Msgf("httpx %v (%v)", addr, server.Addr)
	server.Addr = addr

	return server, nil
}

func (s *Server) MuxX(prefix string) *Mux { return NewServeMux(prefix) }
func (s *Server) Mux() *Mux               { return s.MuxX("") }

func (s *Server) Run() { go s.run() }

func (s *Server) run() {
	protocol := s.GetProtocol()
	s.log.Debug().Msgf("Starting %s server on %s", protocol, s.Addr)

	if s.opts.Https && s.opts.HttpsRedirect {
		rdr, err := s.redirection()
		if err != nil {
			s.log.Error().Err(err).Msg("couldn't init redirection server")
		}
		s.redirect = rdr
		go s.redirect.Run()
	}

	var err error
	if s.opts.Https {
		err = s.ServeTLS(*s.listener, s.opts.HttpsCert, s.opts.HttpsKey)
	} else {
		err = s.Serve(*s.listener)
	}
	switch err {
	case http.ErrServerClosed:
		s.log.Debug().Msgf("%s server was closed", protocol)
		return
	default:
		s.log.Error().Err(err)
	}
}

func (s *Server) Stop() error {
	if s.redirect != nil {
		_ = s.redirect.Stop()
	}
	return s.Server.Close()
}

func (s *Server) GetHost() string { return extractHost(s.Addr) }

func (s *Server) GetProtocol() string {
	protocol := "http"
	if s.opts.Https {
		protocol = "https"
	}
	return protocol
}

func (s *Server) redirection() (*Server, error) {
	address := s.Addr
	if s.opts.HttpsDomain != "" {
		address = s.opts.HttpsDomain
	}
	addr := buildAddress(address, s.opts.Zone, *s.listener)

	srv, err := NewServer(s.opts.HttpsRedirectAddress, func(serv *Server) Handler {
		h := NewServeMux("")
		h.Handle("/", HandlerFunc(func(w ResponseWriter, r *Request) {
			httpsURL := url.URL{Scheme: "https", Host: addr, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
			rdr := httpsURL.String()
			if s.log.GetLevel() < logger.InfoLevel {
				s.log.Debug().
					Str("from", fmt.Sprintf("http://%s%s", r.Host, r.URL.String())).
					Str("to", rdr).
					Msg("Redirect")
			}
			http.Redirect(w, r, rdr, http.StatusFound)
		}))
		if serv.autoCert != nil {
			return serv.autoCert.HTTPHandler(h)
		}
		return h
	},
		WithLogger(s.log),
	)
	s.log.Info().Str("addr", addr).Msg("Start HTTPS redirect server")
	return srv, err
}

func FileServer(dir string) http.Handler { return http.FileServer(http.Dir(dir)) }
