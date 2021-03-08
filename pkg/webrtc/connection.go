package webrtc

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/pion/interceptor"
	. "github.com/pion/webrtc/v3"
)

func NewInterceptedPeerConnection(conf conf.Webrtc, interceptors []interceptor.Interceptor) (*PeerConnection, error) {
	m := &MediaEngine{}
	//if err := m.RegisterDefaultCodecs(); err != nil {
	//	return nil, err
	//}

	if err := RegisterCodecs(m); err != nil {
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

// RegisterCodecs registers the default codecs supported by WebRTC.
func RegisterCodecs(m *MediaEngine) error {
	for _, codec := range []RTPCodecParameters{
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeOpus, ClockRate: 48000, Channels: 2},
			PayloadType:        111,
		},
	} {
		if err := m.RegisterCodec(codec, RTPCodecTypeAudio); err != nil {
			return err
		}
	}

	videoRTCPFeedback := []RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}
	for _, codec := range []RTPCodecParameters{
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeVP8, ClockRate: 90000, RTCPFeedback: videoRTCPFeedback},
			PayloadType:        96,
		},
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeH264, ClockRate: 90000, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        102,
		},
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeH264, ClockRate: 90000, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        108,
		},
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeH264, ClockRate: 90000, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeH264, ClockRate: 90000, RTCPFeedback: videoRTCPFeedback},
			PayloadType:        123,
		},
		{
			RTPCodecCapability: RTPCodecCapability{MimeType: MimeTypeH264, ClockRate: 90000, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        125,
		},
	} {
		if err := m.RegisterCodec(codec, RTPCodecTypeVideo); err != nil {
			return err
		}
	}

	return nil
}
