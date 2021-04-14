package httpx

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"golang.org/x/crypto/acme/autocert"
)

type NetworkServer interface {
}

type Server struct {
	http.Server

	autoCert *autocert.Manager
	opts     Options
}

func NewServer(address string, handler func(serv *Server) http.Handler, options ...Option) *Server {
	opts := &Options{
		Https:         false,
		HttpsRedirect: true,
		// timeouts negate slow / frozen clients
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
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

	return server
}

func (s *Server) Start() {
	// hack: auto open listener on the next free port
	var port int
	address := network.Address(s.Addr)
	if p, err := address.Port(); err == nil {
		port = p
	} else {
		log.Fatalf("error: couldn't extract port from %v", address)
	}

	if s.opts.Https && s.opts.HttpsRedirect {
		log.Printf("Starting HTTP->HTTPS redirection server on %s", s.Addr)
		go NewServer(s.opts.HttpsRedirectAddress, func(serv *Server) http.Handler {
			h := http.NewServeMux()
			h.Handle("/", redirect())
			// do we need this after all?
			if serv.autoCert != nil {
				return serv.autoCert.HTTPHandler(h)
			}
			return h
		}).Start()
	}

	s.start(port)
}

func redirect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusFound)
	})
}

func (s *Server) start(startPort int) {
	protocol := "HTTP"
	if s.opts.Https {
		protocol = "HTTPS"
	}

	endPort := startPort + 1
	if s.opts.PortRoll {
		endPort += 42
	}

	for port := startPort; port < endPort; port++ {
		// !to make it with full address
		s.Addr = ":" + strconv.Itoa(port)
		log.Printf("Starting %s server on %s", protocol, s.Addr)
		var err error
		if s.opts.Https {
			err = s.ListenAndServeTLS(s.opts.HttpsCert, s.opts.HttpsKey)
		} else {
			err = s.ListenAndServe()
		}
		switch err {
		case http.ErrServerClosed:
			log.Printf("%s server was closed", protocol)
			return
		default:
			log.Printf("error: %s", err)
		}

		if s.opts.PortRoll && port+1 == endPort {
			log.Fatalf("error: couldn't find an open port in range %v-%v\n", startPort, endPort)
		}
	}
}
