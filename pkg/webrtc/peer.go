package webrtc

import (
	"fmt"
	"net"
	"sync"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/socket"
	"github.com/pion/interceptor"
	pion "github.com/pion/webrtc/v3"
)

type Peer struct {
	api    *pion.API
	config *pion.Configuration
}

var (
	settingsOnce sync.Once
	settings     pion.SettingEngine
)

func DefaultPeerConnection(conf conf.Webrtc, ts *uint32, log *logger.Logger) (*Peer, error) {
	m := &pion.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	i := &interceptor.Registry{}
	if !conf.DisableDefaultInterceptors {
		if err := pion.RegisterDefaultInterceptors(m, i); err != nil {
			return nil, err
		}
	}
	// todo add re-time only for the main streamer with a game
	i.Add(&ReTimeInterceptor{timestamp: ts})

	settingsOnce.Do(func() {
		customLogger := logger.NewPionLogger(log, conf.LogLevel)
		settingEngine := pion.SettingEngine{LoggerFactory: customLogger}
		if conf.DtlsRole > 0 {
			log.Printf("A custom DTLS role [%v]", conf.DtlsRole)
			if err := settingEngine.SetAnsweringDTLSRole(pion.DTLSRole(conf.DtlsRole)); err != nil {
				panic(err)
			}
		}
		if conf.IceLite {
			settingEngine.SetLite(conf.IceLite)
		}
		if conf.HasPortRange() {
			if err := settingEngine.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
				panic(err)
			}
		}
		if conf.HasSinglePort() {
			l, err := socket.NewSocketPortRoll("udp", conf.SinglePort)
			if err != nil {
				panic(err)
			}
			udp, ok := l.(*net.UDPConn)
			if !ok {
				panic(fmt.Errorf("use of not a UDP socket"))
			}
			settingEngine.SetICEUDPMux(pion.NewICEUDPMux(customLogger, udp))
			log.Info().Msgf("The single port mode is active for %s", udp.LocalAddr())
		}
		if conf.HasIceIpMap() {
			settingEngine.SetNAT1To1IPs([]string{conf.IceIpMap}, pion.ICECandidateTypeHost)
			log.Info().Msgf("The NAT mapping is active for %v", conf.IceIpMap)
		}
		settings = settingEngine
	})

	peerConf := pion.Configuration{ICEServers: []pion.ICEServer{}}
	for _, server := range conf.IceServers {
		peerConf.ICEServers = append(peerConf.ICEServers, pion.ICEServer{
			URLs:       []string{server.Urls},
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	return &Peer{
		api:    pion.NewAPI(pion.WithMediaEngine(m), pion.WithInterceptorRegistry(i), pion.WithSettingEngine(settings)),
		config: &peerConf,
	}, nil
}

func (p *Peer) NewPeer() (*pion.PeerConnection, error) { return p.api.NewPeerConnection(*p.config) }
