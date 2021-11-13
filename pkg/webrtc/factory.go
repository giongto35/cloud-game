package webrtc

import (
	"fmt"
	"net"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/socket"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

type ApiFactory struct {
	api  *webrtc.API
	conf webrtc.Configuration
}

type ModApiFun func(m *webrtc.MediaEngine, i *interceptor.Registry, s *webrtc.SettingEngine)

func NewApiFactory(conf conf.Webrtc, log *logger.Logger, mod ModApiFun) (api *ApiFactory, err error) {
	m := &webrtc.MediaEngine{}
	if err = m.RegisterDefaultCodecs(); err != nil {
		return
	}
	i := &interceptor.Registry{}
	if !conf.DisableDefaultInterceptors {
		if err = webrtc.RegisterDefaultInterceptors(m, i); err != nil {
			return
		}
	}
	customLogger := logger.NewPionLogger(log, conf.LogLevel)
	s := webrtc.SettingEngine{LoggerFactory: customLogger}
	if conf.HasPortRange() {
		if err = s.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
			return
		}
	}
	if conf.HasSinglePort() {
		var l interface{}
		l, err = socket.NewSocketPortRoll("udp", conf.SinglePort)
		if err != nil {
			return
		}
		udp, ok := l.(*net.UDPConn)
		if !ok {
			err = fmt.Errorf("use of not a UDP socket")
			return
		}
		s.SetICEUDPMux(webrtc.NewICEUDPMux(customLogger, udp))
		log.Info().Msgf("The single port mode is active for %s", udp.LocalAddr())
	}
	if conf.HasIceIpMap() {
		s.SetNAT1To1IPs([]string{conf.IceIpMap}, webrtc.ICECandidateTypeHost)
		log.Info().Msgf("The NAT mapping is active for %v", conf.IceIpMap)
	}

	if mod != nil {
		mod(m, i, &s)
	}

	c := webrtc.Configuration{ICEServers: []webrtc.ICEServer{}}
	for _, server := range conf.IceServers {
		c.ICEServers = append(c.ICEServers, webrtc.ICEServer{
			URLs:       []string{server.Urls},
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	return &ApiFactory{
		api:  webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i), webrtc.WithSettingEngine(s)),
		conf: c,
	}, err
}

func (a *ApiFactory) NewPeer() (*webrtc.PeerConnection, error) {
	return a.api.NewPeerConnection(a.conf)
}
