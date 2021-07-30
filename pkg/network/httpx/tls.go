package httpx

import (
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type TLS struct {
	CertManager *autocert.Manager
}

func NewTLSConfig(domain string) *TLS {
	return &TLS{
		CertManager: &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain),
			Cache:      autocert.DirCache("assets/cache"),
			Client:     &acme.Client{DirectoryURL: acme.LetsEncryptURL},
		},
	}
}
