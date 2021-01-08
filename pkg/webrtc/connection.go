package webrtc

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	. "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf conf.Webrtc, interceptors []interceptor.Interceptor) (*PeerConnection, error) {
	m := &MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	i := &interceptor.Registry{}
	if err := RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}
	for _, itc := range interceptors {
		i.Add(itc)
	}

	settingEngine := SettingEngine{}
	if conf.IcePorts.Min > 0 && conf.IcePorts.Max > 0 {
		if err := settingEngine.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
			return nil, err
		}
	}
	if conf.IceIpMap != "" {
		settingEngine.SetNAT1To1IPs([]string{conf.IceIpMap}, ICECandidateTypeHost)
	}

	peerConf := Configuration{ICEServers: []ICEServer{}}
	for _, server := range conf.IceServers {
		peerConf.ICEServers = append(peerConf.ICEServers, ICEServer{
			URLs:       []string{server.Url},
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	api := NewAPI(WithMediaEngine(m), WithInterceptorRegistry(i), WithSettingEngine(settingEngine))
	return api.NewPeerConnection(peerConf)
}
