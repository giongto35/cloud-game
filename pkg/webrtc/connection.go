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

var (
	UDPMuxOnce sync.Once
	udpConn    *net.UDPConn
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

	settingEngine := pion.SettingEngine{}
	if conf.IcePorts.Min > 0 && conf.IcePorts.Max > 0 {
		if err := settingEngine.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
			return nil, err
		}
	}
	if conf.IceIpMap != "" {
		settingEngine.SetNAT1To1IPs([]string{conf.IceIpMap}, pion.ICECandidateTypeHost)
	}
	if conf.SinglePort > 0 {
		UDPMuxOnce.Do(func() {
			// Listen on UDP Port, will be used for all WebRTC traffic
			udpListener, err := net.ListenUDP("udp4",
				&net.UDPAddr{
					//IP:   net.IP{172, 18, 0, 2},
					Port: int(conf.SinglePort),
				},
			)
			if err != nil {
				panic(err)
			}
			_ = udpListener.SetReadBuffer(16_777_216)
			_ = udpListener.SetWriteBuffer(16_777_216)
			log.Printf("---------------------------------")
			log.Printf("Listening for WebRTC traffic at %s\n", udpListener.LocalAddr())
			udpConn = udpListener
		})
		settingEngine.SetICEUDPMux(pion.NewICEUDPMux(nil, udpConn))
		settingEngine.SetNetworkTypes([]pion.NetworkType{pion.NetworkTypeUDP4})
	}

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
			pion.WithSettingEngine(settingEngine),
		),
		config: &peerConf,
	}
	return &conn, nil
}

func (p *PeerConnection) NewConnection() (*pion.PeerConnection, error) {
	return p.api.NewPeerConnection(*p.config)
}
