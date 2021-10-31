package webrtc

import (
	"log"
	"net"
	"sync"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	pion "github.com/pion/webrtc/v3"
)

type PeerConnection struct {
	api    *pion.API
	config *pion.Configuration
}

const udpBufferSize = 16 * 1024 * 1024

var (
	settingsOnce sync.Once
	settings     pion.SettingEngine
)

func DefaultPeerConnection(conf conf.Webrtc, ts *uint32) (*PeerConnection, error) {
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
	i.Add(&ReTimeInterceptor{timestamp: ts})

	settingsOnce.Do(func() {
		settingEngine := pion.SettingEngine{}
		if conf.IcePorts.Min > 0 && conf.IcePorts.Max > 0 {
			if err := settingEngine.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
				panic(err)
			}
		} else {
			if conf.SinglePort > 0 {
				udpListener, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(conf.SinglePort)})
				if err != nil {
					panic(err)
				}
				_ = udpListener.SetReadBuffer(udpBufferSize)
				_ = udpListener.SetWriteBuffer(udpBufferSize)
				log.Printf("Listening for WebRTC traffic at %s\n", udpListener.LocalAddr())
				settingEngine.SetICEUDPMux(pion.NewICEUDPMux(nil, udpListener))
			}
		}
		if conf.IceIpMap != "" {
			settingEngine.SetNAT1To1IPs([]string{conf.IceIpMap}, pion.ICECandidateTypeHost)
		}
		settings = settingEngine
	})

	peerConf := pion.Configuration{ICEServers: []pion.ICEServer{}}
	for _, server := range conf.IceServers {
		peerConf.ICEServers = append(peerConf.ICEServers, pion.ICEServer{
			URLs:       []string{server.Url},
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	conn := PeerConnection{
		api: pion.NewAPI(
			pion.WithMediaEngine(m),
			pion.WithInterceptorRegistry(i),
			pion.WithSettingEngine(settings),
		),
		config: &peerConf,
	}
	return &conn, nil
}

func (p *PeerConnection) NewConnection() (*pion.PeerConnection, error) {
	return p.api.NewPeerConnection(*p.config)
}
