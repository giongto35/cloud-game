package webrtc

import (
	"fmt"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type Peer struct {
	api       *ApiFactory
	conn      *webrtc.PeerConnection
	log       *logger.Logger
	OnMessage func(data []byte)

	aTrack *webrtc.TrackLocalStaticSample
	vTrack *webrtc.TrackLocalStaticSample
	dTrack *webrtc.DataChannel
}

type Decoder func(data string, obj any) error

func New(log *logger.Logger, api *ApiFactory) *Peer { return &Peer{api: api, log: log} }

func (p *Peer) NewCall(vCodec, aCodec string, onICECandidate func(ice any)) (sdp any, err error) {
	if p.IsConnected() {
		return
	}
	p.log.Info().Msg("WebRTC start")
	if p.conn, err = p.api.NewPeer(); err != nil {
		return "", err
	}
	p.conn.OnICECandidate(p.handleICECandidate(onICECandidate))
	// plug in the [video] track (out)
	video, err := newTrack("video", "game-video", vCodec)
	if err != nil {
		return "", err
	}
	if _, err = p.conn.AddTrack(video); err != nil {
		return "", err
	}
	p.vTrack = video
	p.log.Debug().Msgf("Added [%s] track", video.Codec().MimeType)

	// plug in the [audio] track (out)
	audio, err := newTrack("audio", "game-audio", aCodec)
	if err != nil {
		return "", err
	}
	if _, err = p.conn.AddTrack(audio); err != nil {
		return "", err
	}
	p.log.Debug().Msgf("Added [%s] track", audio.Codec().MimeType)
	p.aTrack = audio

	// plug in the [input] data channel (in)
	if err = p.addInputChannel("game-input"); err != nil {
		return "", err
	}
	p.log.Debug().Msg("Added [input/bytes] chan")

	p.conn.OnICEConnectionStateChange(p.handleICEState(func() {
		p.log.Info().Msg("Start streaming")
	}))
	// Stream provider supposes to send offer
	offer, err := p.conn.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	p.log.Info().Msg("Created Offer")

	err = p.conn.SetLocalDescription(offer)
	if err != nil {
		return "", err
	}

	return offer, nil
}

func (p *Peer) SetRemoteSDP(sdp string, decoder Decoder) error {
	var answer webrtc.SessionDescription
	if err := decoder(sdp, &answer); err != nil {
		return err
	}
	if err := p.conn.SetRemoteDescription(answer); err != nil {
		p.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}
	p.log.Debug().Msg("Set Remote Description")
	return nil
}

func (p *Peer) WriteVideo(sample *media.Sample) error { return p.vTrack.WriteSample(*sample) }

func (p *Peer) WriteAudio(sample *media.Sample) error { return p.aTrack.WriteSample(*sample) }

func newTrack(id string, label string, codec string) (*webrtc.TrackLocalStaticSample, error) {
	codec = strings.ToLower(codec)
	var mime string
	switch id {
	case "audio":
		switch codec {
		case "opus":
			mime = webrtc.MimeTypeOpus
		}
	case "video":
		switch codec {
		case "h264":
			mime = webrtc.MimeTypeH264
		case "vpx", "vp8":
			mime = webrtc.MimeTypeVP8
		}
	}
	if mime == "" {
		return nil, fmt.Errorf("unsupported codec %s:%s", id, codec)
	}
	return webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mime}, id, label)
}

func (p *Peer) handleICECandidate(callback func(any)) func(*webrtc.ICECandidate) {
	return func(ice *webrtc.ICECandidate) {
		// ICE gathering finish condition
		if ice == nil {
			callback(nil)
			p.log.Debug().Msg("ICE gathering was complete probably")
			return
		}
		candidate := ice.ToJSON()
		p.log.Debug().Str("candidate", candidate.Candidate).Msg("ICE")
		callback(&candidate)
	}
}

func (p *Peer) handleICEState(onConnect func()) func(webrtc.ICEConnectionState) {
	return func(state webrtc.ICEConnectionState) {
		p.log.Debug().Str(".state", state.String()).Msg("ICE")
		switch state {
		case webrtc.ICEConnectionStateChecking:
			// nothing
		case webrtc.ICEConnectionStateConnected:
			onConnect()
		case webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed,
			webrtc.ICEConnectionStateDisconnected:
			p.Disconnect()
		default:
			p.log.Debug().Msg("ICE state is not handled!")
		}
	}
}

func (p *Peer) AddCandidate(candidate string, decoder Decoder) error {
	var iceCandidate webrtc.ICECandidateInit
	if err := decoder(candidate, &iceCandidate); err != nil {
		return err
	}
	if err := p.conn.AddICECandidate(iceCandidate); err != nil {
		return err
	}
	p.log.Debug().Str("candidate", iceCandidate.Candidate).Msg("Ice")
	return nil
}

func (p *Peer) Disconnect() {
	if p.conn == nil {
		return
	}
	if p.conn.ConnectionState() < webrtc.PeerConnectionStateDisconnected {
		_ = p.conn.Close()
	}
	p.conn = nil
	p.log.Debug().Msg("WebRTC stop")
}

func (p *Peer) IsConnected() bool {
	return p.conn != nil && p.conn.ConnectionState() == webrtc.PeerConnectionStateConnected
}

func (p *Peer) SendMessage(data []byte) { _ = p.dTrack.Send(data) }

// addInputChannel creates a new WebRTC data channel for user input.
// Default params -- ordered: true, negotiated: false.
func (p *Peer) addInputChannel(label string) error {
	ch, err := p.conn.CreateDataChannel(label, nil)
	if err != nil {
		return err
	}
	ch.OnOpen(func() {
		p.log.Debug().Str("label", ch.Label()).Uint16("id", *ch.ID()).Msg("Data channel [input] opened")
	})
	ch.OnError(p.logx)
	ch.OnMessage(func(mess webrtc.DataChannelMessage) {
		if len(mess.Data) == 0 {
			return
		}
		// echo string messages (e.g. ping/pong)
		if mess.IsString {
			p.logx(ch.Send(mess.Data))
			return
		}
		if p.OnMessage != nil {
			p.OnMessage(mess.Data)
		}
	})
	p.dTrack = ch
	ch.OnClose(func() { p.log.Debug().Msg("Data channel [input] has been closed") })
	return nil
}

func (p *Peer) logx(err error) { p.log.Error().Err(err) }
