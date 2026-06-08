package webrtc

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type Peer struct {
	api       *ApiFactory
	conn      *webrtc.PeerConnection
	log       *logger.Logger
	OnMessage func(data []byte)

	a *webrtc.TrackLocalStaticSample
	v *webrtc.TrackLocalStaticSample
	d *webrtc.DataChannel

	// flags
	makingOffer                  bool
	ignoreOffer                  bool
	isSettingRemoteAnswerPending bool
	polite                       bool

	signaller func(ice, sdp *string)
}

var samplePool sync.Pool

func New(log *logger.Logger, api *ApiFactory) *Peer {
	// hide directions (->) and clients (w, c, ...)
	custom := log.Extend(
		log.With().
			Str(logger.DirectionField, "").
			Str(logger.ClientField, ""),
	)
	return &Peer{api: api, log: custom}
}

func (p *Peer) NewConnection(vCodec, aCodec string, signal func(*string, *string)) (err error) {
	p.log.Debug().Msg("rtc start")

	pc := p.conn
	p.signaller = signal

	if pc != nil && pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return
	}

	if p.conn, err = p.api.NewPeer(); err != nil {
		return
	}
	pc = p.conn

	pc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		p.log.Debug().
			Str("state", pcs.String()).
			Msg("rtc [connection] state change")
		if pcs == webrtc.PeerConnectionStateConnected {
			p.log.Info().
				Str(logger.DirectionField, "rtc").
				Str(logger.ClientField, "").
				Msg("(connected)")
		}
	})
	pc.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			signal(new(string), nil)
			p.log.Debug().Msg("rtc [ice] gathering (complete)")
			return
		}
		candidate := ice.ToJSON()
		p.log.Debug().Str("local", candidate.Candidate).Msg("rtc [ice] candidate")

		encoded, err := toJson(candidate)
		if err != nil {
			p.log.Error().Err(err).Msg("rtc [ice] candidate encoding failed")
			return
		}

		signal(&encoded, nil)
	})
	pc.OnSignalingStateChange(func(ss webrtc.SignalingState) {
		p.log.Debug().Str("state", ss.String()).Msg("rtc [signal] state change")
	})
	pc.OnICEConnectionStateChange(p.handleICEState(func() {}))
	pc.OnDataChannel(func(ch *webrtc.DataChannel) {
		p.log.Debug().Msgf("rtc [chan] [%s] remote", ch.Label())
	})

	p.makingOffer = false

	pc.OnNegotiationNeeded(func() {
		p.log.Debug().Msg("rtc [negotiation] needed")
		defer func() { p.makingOffer = false }()

		p.makingOffer = true
		offer, err := p.Offer()
		if err != nil {
			p.log.Error().Err(err).Msg("rtc [negotiation] failed")
			return
		}

		// if p.conn.SignalingState() != webrtc.SignalingStateStable {
		// p.log.Debug().Msg("rtc [negotiation] waiting for signaling state stable")
		// return
		// }

		sdp, err := toJson(offer)
		if err != nil {
			p.log.Error().Err(err).Msg("rtc [negotiation] failed")
			return
		}

		signal(nil, &sdp)
	})

	if p.v, err = NewTrack("video", "video", vCodec); err != nil {
		return err
	}
	p.log.Debug().Msgf("rtc [media] added [%s] track", p.v.Codec().MimeType)

	if p.a, err = NewTrack("audio", "audio", aCodec); err != nil {
		return err
	}
	p.log.Debug().Msgf("rtc [media] added [%s] track", p.a.Codec().MimeType)

	sendOnly := webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}

	if _, err = p.conn.AddTransceiverFromTrack(p.v, sendOnly); err != nil {
		panic(err)
	}
	if _, err = p.conn.AddTransceiverFromTrack(p.a, sendOnly); err != nil {
		panic(err)
	}

	for _, rtpSender := range p.conn.GetSenders() {
		go processRTCP(rtpSender)
	}

	id := uint16(0)
	negotiated := true
	ordered := false
	maxRetransmits := uint16(0)

	p.d, err = p.AddChannel("data", &webrtc.DataChannelInit{
		ID:             &id,
		Negotiated:     &negotiated,
		Ordered:        &ordered,
		MaxRetransmits: &maxRetransmits,
	},
		func(data []byte) {
			if len(data) == 0 || p.OnMessage == nil {
				return
			}
			p.OnMessage(data)
		})
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) Offer() (*webrtc.SessionDescription, error) {
	opts := webrtc.OfferAnswerOptions{ICETricklingSupported: true}

	sdp, err := p.conn.CreateOffer(&webrtc.OfferOptions{OfferAnswerOptions: opts})
	if err != nil {
		return nil, err
	}
	if err = p.conn.SetLocalDescription(sdp); err != nil {
		return nil, err
	}
	p.log.Debug().Str("type", sdp.Type.String()).Msg("rtc [sdp] set (local)")

	return &sdp, nil
}

func (p *Peer) SendAudio(dat []byte, dur int32) {
	if err := p.send(dat, int64(dur), p.a.WriteSample); err != nil {
		p.log.Error().Err(err).Send()
	}
}

func (p *Peer) SendVideo(data []byte, dur int32) {
	if err := p.send(data, int64(dur), p.v.WriteSample); err != nil {
		p.log.Error().Err(err).Send()
	}
}

func (p *Peer) SendData(data []byte) { _ = p.d.Send(data) }

func (p *Peer) send(data []byte, duration int64, fn func(media.Sample) error) error {
	sample, _ := samplePool.Get().(*media.Sample)
	if sample == nil {
		sample = new(media.Sample)
	}
	sample.Data = data
	sample.Duration = time.Duration(duration)
	err := fn(*sample)
	if err != nil {
		return err
	}
	samplePool.Put(sample)
	return nil
}

func (p *Peer) SetDescription(sdp string) error {
	var answer webrtc.SessionDescription
	if err := fromJson(sdp, &answer); err != nil {
		return err
	}

	readyForOffer := !p.makingOffer &&
		(p.conn.SignalingState() == webrtc.SignalingStateStable || p.isSettingRemoteAnswerPending)

	offerCollision := answer.Type == webrtc.SDPTypeOffer && !readyForOffer

	ignoreOffer := !p.polite && offerCollision
	if ignoreOffer {
		return nil
	}
	p.isSettingRemoteAnswerPending = answer.Type == webrtc.SDPTypeAnswer
	if err := p.conn.SetRemoteDescription(answer); err != nil {
		p.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}
	p.isSettingRemoteAnswerPending = false
	if answer.Type == webrtc.SDPTypeOffer {
		err := p.conn.SetLocalDescription(answer)
		if err != nil {
			p.log.Error().Err(err).Msg("Set remote description from peer failed")
			return err
		}

		sdp, err := toJson(answer)
		if err != nil {
			return err
		}

		p.signaller(nil, &sdp)
	}
	p.log.Debug().Str("type", answer.Type.String()).Msg("rtc [sdp] set (remote)")

	return nil
}

func NewTrack(id string, label string, codec string) (*webrtc.TrackLocalStaticSample, error) {
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
		case "vp9":
			mime = webrtc.MimeTypeVP9
		}
	}
	if mime == "" {
		return nil, fmt.Errorf("unsupported codec %s:%s", id, codec)
	}
	return webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mime}, id, label)
}

func (p *Peer) handleICEState(onConnect func()) func(webrtc.ICEConnectionState) {
	return func(state webrtc.ICEConnectionState) {
		p.log.Debug().Str("state", state.String()).Msg("rtc [ice] connection state change")
		switch state {
		case webrtc.ICEConnectionStateChecking:
			// nothing
		case webrtc.ICEConnectionStateConnected:
			onConnect()
		case webrtc.ICEConnectionStateFailed:
			p.log.Error().Msgf("WebRTC connection fail! connection: %v, ice: %v, gathering: %v, signalling: %v",
				p.conn.ConnectionState(), p.conn.ICEConnectionState(), p.conn.ICEGatheringState(),
				p.conn.SignalingState())
			// make ICE restart
		case webrtc.ICEConnectionStateDisconnected:
		case webrtc.ICEConnectionStateClosed:
			p.Disconnect()
		default:
			p.log.Debug().Msg("ICE state is not handled!")
		}
	}
}

func (p *Peer) AddCandidate(candidate string) error {
	var iceCandidate webrtc.ICECandidateInit
	if err := fromJson(candidate, &iceCandidate); err != nil {
		return err
	}
	p.log.Debug().Str("remote", iceCandidate.Candidate).Msg("rtc [ice] candidate")

	err := p.conn.AddICECandidate(iceCandidate)
	if p.ignoreOffer {
		err = nil
	}
	return err
}

func (p *Peer) AddChannel(label string, conf *webrtc.DataChannelInit, onMessage func([]byte)) (*webrtc.DataChannel, error) {
	config := conf
	if conf == nil {
		ordered := false
		maxRetransmits := uint16(0)
		config = &webrtc.DataChannelInit{Ordered: &ordered, MaxRetransmits: &maxRetransmits}
	}

	ch, err := p.conn.CreateDataChannel(label, config)
	if err != nil {
		return nil, err
	}

	ch.OnOpen(func() {
		p.log.Debug().
			Uint16("id", *ch.ID()).
			Msgf("rtc [chan] [%v] opened", ch.Label())
	})
	ch.OnMessage(func(m webrtc.DataChannelMessage) { onMessage(m.Data) })
	ch.OnError(p.logx)
	ch.OnClose(func() {
		p.log.Debug().Msgf("rtc [chan] [%v] has been closed", ch.Label())
	})
	p.log.Debug().Msgf("rtc [chan] [%v] added", label)

	return ch, nil
}

func (p *Peer) Disconnect() {
	if p.conn == nil {
		return
	}
	if p.conn.ConnectionState() < webrtc.PeerConnectionStateDisconnected {
		// ignore this due to DTLS fatal: conn is closed
		_ = p.conn.Close()
	}
	p.log.Debug().Msg("rtc stopped")
}

// Read incoming RTCP packets
// Before these packets are returned they are processed by interceptors. For things
// like NACK this needs to be called.
func processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}

func fromJson(data string, obj any) error {
	return json.Unmarshal([]byte(data), obj)
}

func toJson(data any) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Peer) logx(err error) { p.log.Error().Err(err) }
