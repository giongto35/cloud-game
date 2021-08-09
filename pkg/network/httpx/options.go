package httpx

import (
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
)

type (
	Options struct {
		Https                bool
		HttpsRedirect        bool
		HttpsRedirectAddress string
		HttpsCert            string
		HttpsKey             string
		HttpsDomain          string
		PortRoll             bool
		IdleTimeout          time.Duration
		ReadTimeout          time.Duration
		WriteTimeout         time.Duration
		Zone                 string
	}
	Option func(*Options)
)

func (o *Options) override(options ...Option) {
	for _, opt := range options {
		opt(o)
	}
}

func (o *Options) IsAutoHttpsCert() bool { return !(o.HttpsCert != "" && o.HttpsKey != "") }

func HttpsRedirect(redirect bool) Option {
	return func(opts *Options) { opts.HttpsRedirect = redirect }
}

//func Https(is bool) Option                { return func(opts *Options) { opts.Https = is } }
//func HttpsCert(cert string) Option        { return func(opts *Options) { opts.HttpsCert = cert } }
//func HttpsKey(key string) Option          { return func(opts *Options) { opts.HttpsKey = key } }
//func HttpsDomain(domain string) Option    { return func(opts *Options) { opts.HttpsDomain = domain } }
//func IdleTimeout(t time.Duration) Option  { return func(opts *Options) { opts.IdleTimeout = t } }
//func ReadTimeout(t time.Duration) Option  { return func(opts *Options) { opts.ReadTimeout = t } }
//func WriteTimeout(t time.Duration) Option { return func(opts *Options) { opts.WriteTimeout = t } }

func WithPortRoll(roll bool) Option { return func(opts *Options) { opts.PortRoll = roll } }
func WithZone(zone string) Option   { return func(opts *Options) { opts.Zone = zone } }
func WithServerConfig(conf shared.Server) Option {
	return func(opts *Options) {
		opts.Https = conf.Https
		opts.HttpsCert = conf.Tls.HttpsCert
		opts.HttpsKey = conf.Tls.HttpsKey
		opts.HttpsDomain = conf.Tls.Domain
		opts.HttpsRedirectAddress = conf.Address
	}
}
