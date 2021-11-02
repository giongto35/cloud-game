package webrtc

import "github.com/giongto35/cloud-game/v2/pkg/config/encoder"

type Webrtc struct {
	DisableDefaultInterceptors bool
	IceServers                 []IceServer
	IcePorts                   struct {
		Min uint16
		Max uint16
	}
	IceIpMap   string
	SinglePort int
}

type IceServer struct {
	Url        string
	Username   string
	Credential string
}

type Config struct {
	Encoder encoder.Encoder
	Webrtc  Webrtc
}
