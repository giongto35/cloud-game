package webrtc

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	. "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf conf.Webrtc, ics []interceptor.Interceptor, vCodec string) (*PeerConnection, error) {
	m := &MediaEngine{}
	//if err := m.RegisterDefaultCodecs(); err != nil {
	//	return nil, err
	//}

	if err := RegisterCodecs(m, vCodec); err != nil {
		return nil, err
	}

	i := &interceptor.Registry{}
	if err := RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}
	for _, itc := range ics {
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

// RegisterCodecs registers the default codecs supported by WebRTC.
func RegisterCodecs(m *MediaEngine, vCodec string) error {
	audioRTPCodecParameters := []RTPCodecParameters{
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeOpus, ClockRate: 48000, Channels: 2},
			PayloadType:        111,
		},
	}
	for _, codec := range audioRTPCodecParameters {
		if err := m.RegisterCodec(codec, RTPCodecTypeAudio); err != nil {
			return err
		}
	}

	videoRTCPFeedback := []RTCPFeedback{
		{"goog-remb", ""},
		{"ccm", "fir"},
		{"nack", ""},
		{"nack", "pli"},
	}
	video := RTPCodecCapability{MimeType: vCodec, ClockRate: 90000, RTCPFeedback: videoRTCPFeedback}
	var videoRTPCodecParameters []RTPCodecParameters
	if vCodec == MimeTypeH264 {
		videoRTPCodecParameters = []RTPCodecParameters{
			{RTPCodecCapability: RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				//SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
			}, PayloadType: 102},
			{RTPCodecCapability: RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;profile-level-id=42e01f",
			}, PayloadType: 108},
			{RTPCodecCapability: video, PayloadType: 123},
			{RTPCodecCapability: RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
			}, PayloadType: 125},
			{RTPCodecCapability: RTPCodecCapability{
				MimeType: video.MimeType, ClockRate: video.ClockRate, RTCPFeedback: video.RTCPFeedback,
				SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f",
			}, PayloadType: 127},
		}
	} else {
		videoRTPCodecParameters = []RTPCodecParameters{
			{RTPCodecCapability: video, PayloadType: 96},
		}
	}

	for _, codec := range videoRTPCodecParameters {
		if err := m.RegisterCodec(codec, RTPCodecTypeVideo); err != nil {
			return err
		}
	}

	return nil
}
