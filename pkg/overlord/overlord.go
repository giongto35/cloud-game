package overlord

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/pkg/monitoring"
	"github.com/golang/glog"

	"golang.org/x/crypto/acme/autocert"
)

type Overlord struct {
	ctx context.Context
	cfg Config

	monitoringServer *monitoring.ServerMonitoring
}

func New(ctx context.Context, cfg Config) *Overlord {
	return &Overlord{
		ctx: ctx,
		cfg: cfg,

		monitoringServer: monitoring.NewServerMonitoring(cfg.MonitoringConfig),
	}
}

func (o *Overlord) Run() error {
	go o.initializeOverlord()
	go o.RunMonitoringServer()
	return nil
}

func (o *Overlord) RunMonitoringServer() {
	glog.Infoln("Starting monitoring server for overlord")
	err := o.monitoringServer.Run()
	if err != nil {
		glog.Errorf("Failed to start monitoring server, reason %s", err)
	}
}

func (o *Overlord) Shutdown() {
	if err := o.monitoringServer.Shutdown(o.ctx); err != nil {
		glog.Errorln("Failed to shutdown monitoring server")
	}
}

func makeHTTPToHTTPSRedirectServer() *http.Server {
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	}
	server := http.NewServeMux()
	server.HandleFunc("/", handleRedirect)
	return server
}

// initializeOverlord setup an overlord server
func (o *Overlord) initializeOverlord() {
	overlord := NewServer(o.cfg)

	hostPolicy := func(ctx context.Context, host string) error {
		// Note: change to your real host
		allowedHost := "www.cloudretro.io"
		if host == allowedHost {
			return nil
		}
		return fmt.Errorf("acme/autocert: only %s host is allowed", allowedHost)
	}
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Cache:      autocert.DirCache("certs"),
	}
	fmt.Println("HAHA")
	mux := http.NewServeMux()
	mux.HandleFunc("/", overlord.GetWeb)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web"))))

	fmt.Println("HIHI")
	// browser facing port
	go func() {
		mux.HandleFunc("/ws", overlord.WS)
	}()
	mux.HandleFunc("/wso", overlord.WSO)

	s := &http.Server{
		Addr:    ":443",
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	fmt.Println("HOHO")
	// worker facing port
	log.Println("Listening at port: localhost:8000")
	go func() {
		err := http.ListenAndServe(":8000", certManager.HTTPHandler(nil))
		// Print err if overlord cannot launch
		if err != nil {
			log.Fatal(err)
		}
	}()

	s.ListenAndServeTLS("", "")
}

//func makeHTTPServer() *http.Server {
//mux := &http.ServeMux{}
//mux.HandleFunc("/", handleIndex)
//return makeServerFromMux(mux)
//}
