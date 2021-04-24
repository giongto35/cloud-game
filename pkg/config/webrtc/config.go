package webrtc

import "github.com/giongto35/cloud-game/v2/pkg/config/encoder"

type Webrtc struct {
	DisableDefaultInterceptors bool
	IceServers                 []IceServer
	IcePorts                   struct {
		Min uint16
		Max uint16
	}
	IceIpMap string
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
