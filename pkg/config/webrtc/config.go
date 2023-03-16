package webrtc

import (
	"log"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/config"
)

type Webrtc struct {
	DisableDefaultInterceptors bool
	DtlsRole                   byte
	IceServers                 []IceServer
	IcePorts                   struct {
		Min uint16
		Max uint16
	}
	IceIpMap   string
	IceLite    bool
	SinglePort int
	LogLevel   int
}

type IceServer struct {
	Urls       string `json:"urls,omitempty"`
	Username   string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}

func (w *Webrtc) HasDtlsRole() bool   { return w.DtlsRole > 0 }
func (w *Webrtc) HasPortRange() bool  { return w.IcePorts.Min > 0 && w.IcePorts.Max > 0 }
func (w *Webrtc) HasSinglePort() bool { return w.SinglePort > 0 }
func (w *Webrtc) HasIceIpMap() bool   { return w.IceIpMap != "" }

func (w *Webrtc) AddIceServersEnv() {
	cfg := Webrtc{IceServers: []IceServer{{}, {}, {}, {}, {}}}
	_ = config.LoadConfigEnv(&cfg)
	for i, ice := range cfg.IceServers {
		if ice.Urls == "" {
			continue
		}
		if strings.HasPrefix(ice.Urls, "turn:") || strings.HasPrefix(ice.Urls, "turns:") {
			if ice.Username == "" || ice.Credential == "" {
				log.Fatalf("TURN or TURNS servers should have both username and credential: %+v", ice)
			}
		}
		if i > len(w.IceServers)-1 {
			w.IceServers = append(w.IceServers, ice)
		} else {
			w.IceServers[i] = ice
		}
	}
}
