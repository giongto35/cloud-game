package httpx

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/service"
	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	http.Server
	service.RunnableService

	autoCert *autocert.Manager
	opts     Options

	listener *Listener
	redirect *Server
}

func NewServer(address string, handler func(*Server) http.Handler, options ...Option) (*Server, error) {
	opts := &Options{
		Https:         false,
		HttpsRedirect: true,
		IdleTimeout:   120 * time.Second,
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
	}
	opts.override(options...)

	server := &Server{
		Server: http.Server{
			Addr:         address,
			IdleTimeout:  opts.IdleTimeout,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
		},
		opts: *opts,
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
		log.Printf("Warning! Empty server address has been changed to %v", addr)
	}
	listener, err := NewListener(addr, server.opts.PortRoll)
	if err != nil {
		return nil, err
	}
	server.listener = listener

	addr = buildAddress(server.Addr, opts.Zone, *listener)
	log.Printf("[server] address was set to %v (%v)", addr, server.Addr)
	server.Addr = addr

	return server, nil
}

func (s *Server) Run() {
	protocol := s.GetProtocol()
	log.Printf("Starting %s server on %s", protocol, s.Addr)

	if s.opts.Https && s.opts.HttpsRedirect {
		rdr, err := s.redirection()
		if err != nil {
			log.Fatalf("couldn't init redirection server: %v", err)
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
		log.Printf("%s server was closed", protocol)
		return
	default:
		log.Printf("error: %s", err)
	}
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	if s.redirect != nil {
		err = s.redirect.Shutdown(ctx)
	}
	err = s.Server.Shutdown(ctx)
	return
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

	srv, err := NewServer(s.opts.HttpsRedirectAddress, func(serv *Server) http.Handler {
		h := http.NewServeMux()

		h.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpsURL := url.URL{Scheme: "https", Host: addr, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
			rdr := httpsURL.String()
			log.Printf("Redirect: http://%s%s -> %s", r.Host, r.URL.String(), rdr)
			http.Redirect(w, r, rdr, http.StatusFound)
		}))

		// do we need this after all?
		if serv.autoCert != nil {
			return serv.autoCert.HTTPHandler(h)
		}
		return h
	})
	log.Printf("Starting HTTP->HTTPS redirection server on %s", addr)
	return srv, err
}
