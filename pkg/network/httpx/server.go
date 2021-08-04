package httpx

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/service"
	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	http.Server
	service.RunnableService

	autoCert *autocert.Manager
	opts     Options

	listener *net.Listener
	redirect *Server
}

func NewServer(address string, handler func(serv *Server) http.Handler, options ...Option) (*Server, error) {
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
		server.autoCert = NewTLSConfig(opts.HttpsDomain).CertManager
		server.TLSConfig = server.autoCert.TLSConfig()
	}

	addr := server.Addr
	if server.Addr == "" {
		addr = ":http"
		if opts.Https {
			addr = ":https"
		}
		log.Printf("Warning! Empty server address has been changed to %v", server.Addr)
	}
	listener, err := NewListener(addr, server.opts.PortRoll)
	if err != nil {
		return nil, err
	}
	server.listener = &listener

	addr = mergeAddresses(server.Addr, listener)
	log.Printf("[server] address was set to %v (%v)", addr, server.Addr)
	server.Addr = addr

	return server, nil
}

func (s *Server) Run() {
	if s == nil || s.listener == nil {
		return
	}

	protocol := "HTTP"
	if s.opts.Https {
		protocol = "HTTPS"
	}

	log.Printf("Starting %s server on %s", protocol, s.Addr)

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

	if s.opts.Https && s.opts.HttpsRedirect {
		s.redirect = s.redirection()
		go s.redirect.Run()
	}
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	if s == nil {
		return
	}
	if s.redirect != nil {
		err = s.redirect.Shutdown(ctx)
	}
	err = s.Server.Shutdown(ctx)
	return
}

func (s *Server) redirection() *Server {
	// !to handle error
	serv, _ := NewServer(s.opts.HttpsRedirectAddress, func(serv *Server) http.Handler {
		h := http.NewServeMux()
		h.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusFound)
		}))
		// do we need this after all?
		if serv.autoCert != nil {
			return serv.autoCert.HTTPHandler(h)
		}
		return h
	})
	log.Printf("Starting HTTP->HTTPS redirection server on %s", serv.Addr)

	return serv
}
