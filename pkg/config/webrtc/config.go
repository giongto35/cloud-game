package webrtc

import (
	"log"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/encoder"
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

type Config struct {
	Encoder encoder.Encoder
	Webrtc  Webrtc
}

func (w *Webrtc) AddIceServersEnv() {
	cfg := Config{Webrtc: Webrtc{IceServers: []IceServer{{}, {}, {}, {}, {}}}}
	_ = config.LoadConfigEnv(&cfg)
	for i, ice := range cfg.Webrtc.IceServers {
		if ice.Url == "" {
			continue
		}
		if strings.HasPrefix(ice.Url, "turn:") || strings.HasPrefix(ice.Url, "turns:") {
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
