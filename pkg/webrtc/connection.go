package webrtc

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	pion "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf conf.Webrtc, ics []interceptor.Interceptor) (*pion.PeerConnection, error) {
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
	for _, itc := range ics {
		i.Add(itc)
	}

	settingEngine := pion.SettingEngine{}
	if conf.IcePorts.Min > 0 && conf.IcePorts.Max > 0 {
		if err := settingEngine.SetEphemeralUDPPortRange(conf.IcePorts.Min, conf.IcePorts.Max); err != nil {
			return nil, err
		}
	}
	if conf.IceIpMap != "" {
		settingEngine.SetNAT1To1IPs([]string{conf.IceIpMap}, pion.ICECandidateTypeHost)
	}

	peerConf := pion.Configuration{ICEServers: []pion.ICEServer{}}
	for _, server := range conf.IceServers {
		peerConf.ICEServers = append(peerConf.ICEServers, pion.ICEServer{
			URLs:       []string{server.Url},
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	api := pion.NewAPI(
		pion.WithMediaEngine(m),
		pion.WithInterceptorRegistry(i),
		pion.WithSettingEngine(settingEngine),
	)
	return api.NewPeerConnection(peerConf)
}
