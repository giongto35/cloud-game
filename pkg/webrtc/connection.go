package webrtc

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	pion "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf conf.Webrtc, ics []interceptor.Interceptor, vCodec string) (*pion.PeerConnection, error) {
	m := &pion.MediaEngine{}
	//if err := m.RegisterDefaultCodecs(); err != nil {
	//	return nil, err
	//}

	if err := RegisterCodecs(m, vCodec); err != nil {
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

// RegisterCodecs registers the default codecs supported by WebRTC.
func RegisterCodecs(m *pion.MediaEngine, vCodec string) error {
	audioRTPCodecParameters := []pion.RTPCodecParameters{
		{
			RTPCodecCapability: pion.RTPCodecCapability{MimeType: pion.MimeTypeOpus, ClockRate: 48000, Channels: 2},
			PayloadType:        111,
		},
	}
	for _, codec := range audioRTPCodecParameters {
		if err := m.RegisterCodec(codec, pion.RTPCodecTypeAudio); err != nil {
			return err
		}
	}

	videoRTCPFeedback := []pion.RTCPFeedback{
		{"goog-remb", ""},
		{"ccm", "fir"},
		{"nack", ""},
		{"nack", "pli"},
	}
	video := pion.RTPCodecCapability{MimeType: vCodec, ClockRate: 90000, RTCPFeedback: videoRTCPFeedback}
	var videoRTPCodecParameters []pion.RTPCodecParameters
	if vCodec == pion.MimeTypeH264 {
		videoRTPCodecParameters = []pion.RTPCodecParameters{
			{RTPCodecCapability: pion.RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				//SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
			}, PayloadType: 102},
			{RTPCodecCapability: pion.RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;profile-level-id=42e01f",
			}, PayloadType: 108},
			{RTPCodecCapability: video, PayloadType: 123},
			{RTPCodecCapability: pion.RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
			}, PayloadType: 125},
			{RTPCodecCapability: pion.RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f",
			}, PayloadType: 127},
		}
	} else {
		videoRTPCodecParameters = []pion.RTPCodecParameters{
			{RTPCodecCapability: video, PayloadType: 96},
		}
	}

	for _, codec := range videoRTPCodecParameters {
		if err := m.RegisterCodec(codec, pion.RTPCodecTypeVideo); err != nil {
			return err
		}
	}

	return nil
}
