package httpx

import "golang.org/x/crypto/acme/autocert"

type TLS struct {
	CertManager *autocert.Manager
}

func NewTLSConfig(host string) *TLS {
	tls := TLS{
		CertManager: &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache("assets/cache"),
		},
	}
	if host != "" {
		tls.CertManager.HostPolicy = autocert.HostWhitelist(host)
	}
	return &tls
}
