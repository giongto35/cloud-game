package webrtc

import (
	"cmp"
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
	api *ApiFactory
	c   *webrtc.PeerConnection
	log *logger.Logger

	a *webrtc.TrackLocalStaticSample
	v *webrtc.TrackLocalStaticSample
	d *webrtc.DataChannel

	onMessage func(data []byte)
}

var samplePool sync.Pool

var DefaultOfferAnswerOptions = webrtc.OfferAnswerOptions{ICETricklingSupported: true}
var DefaultOfferOptions = webrtc.OfferOptions{OfferAnswerOptions: DefaultOfferAnswerOptions}
var DefaultAnswerOptions = webrtc.AnswerOptions{OfferAnswerOptions: DefaultOfferAnswerOptions}
var DefaultChannelParams = webrtc.DataChannelInit{Ordered: new(false), MaxRetransmits: new(uint16)}
var DefaultDataChannelParams = webrtc.DataChannelInit{
	ID:             new(uint16),
	Negotiated:     new(true),
	Ordered:        new(false),
	MaxRetransmits: new(uint16),
}

func New(log *logger.Logger, api *ApiFactory) *Peer {
	// hide directions (->) and clients (w, c, ...)
	custom := log.Extend(
		log.With().
			Str(logger.DirectionField, "").
			Str(logger.ClientField, ""),
	)
	return &Peer{api: api, log: custom}
}

func (p *Peer) NewConnection(vCodec, aCodec string, signaller func(ice *string)) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("couldn't create webrtc seesion: %w", err)
		}
	}()

	if p.c != nil && p.c.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return fmt.Errorf("peer connected already")
	}

	p.log.Debug().Msg("rtc start")

	if p.c, err = p.api.NewPeer(); err != nil {
		return
	}

	p.c.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		p.log.Debug().Str("state", pcs.String()).Msg("rtc [connection] state change")
		if pcs == webrtc.PeerConnectionStateConnected {
			p.log.Info().Str(logger.DirectionField, "rtc").Str(logger.ClientField, "").Msg("(connected)")
		}
	})
	p.c.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			signaller(new(""))
			p.log.Debug().Msg("rtc [ice] gathering (complete)")
			return
		}
		candidate := ice.ToJSON()
		p.log.Debug().Str("local", candidate.Candidate).Msg("rtc [ice] candidate")

		encoded, err := toJson(candidate)
		if err != nil {
			p.log.Error().Err(err).Msg("rtc [ice] candidate marshal")
			return
		}
		signaller(&encoded)
	})

	p.c.OnSignalingStateChange(func(ss webrtc.SignalingState) {
		p.log.Debug().Str("state", ss.String()).Msg("rtc [signal] state change")
	})

	p.c.OnICEConnectionStateChange(p.handleICEState)

	p.c.OnDataChannel(func(ch *webrtc.DataChannel) { p.log.Debug().Msgf("rtc [chan] [%s] remote", ch.Label()) })

	if err = p.InitMedia(vCodec, aCodec); err != nil {
		return
	}

	onData := func(data []byte) {
		if len(data) != 0 && p.onMessage != nil {
			p.onMessage(data)
		}
	}

	if p.d, err = p.Channel("data", &DefaultDataChannelParams, onData); err != nil {
		return err
	}

	p.c.OnNegotiationNeeded(func() { p.log.Debug().Msg("rtc [negotiation] needed") })

	return nil
}

func (p *Peer) OfferAnswer(offer bool) (string, error) {
	var sdp webrtc.SessionDescription
	var err error

	if offer {
		sdp, err = p.c.CreateOffer(&DefaultOfferOptions)
	} else {
		sdp, err = p.c.CreateAnswer(&DefaultAnswerOptions)
	}
	if err != nil {
		return "", err
	}

	if err = p.c.SetLocalDescription(sdp); err != nil {
		return "", err
	}

	encoded, err := toJson(sdp)
	if err != nil {
		return "", err
	}

	p.log.Debug().Str("type", sdp.Type.String()).Msg("rtc [sdp] (local)")

	return encoded, nil
}

func (p *Peer) HandleSignal(ice, sdp *string) error {
	if ice != nil {
		candidate, err := fromJson[webrtc.ICECandidateInit](*ice)
		if err != nil {
			return err
		}
		p.log.Debug().Str("remote", candidate.Candidate).Msg("rtc [ice] candidate")
		return p.c.AddICECandidate(candidate)
	}
	if sdp != nil {
		answer, err := fromJson[webrtc.SessionDescription](*sdp)
		if err != nil {
			return err
		}
		p.log.Debug().Str("type", answer.Type.String()).Msg("rtc [sdp] (remote)")
		return p.c.SetRemoteDescription(answer)
	}
	return nil
}

func newTrack(id string, label string, codec string) (*webrtc.TrackLocalStaticSample, error) {
	codec = strings.ToLower(codec)
	var mime string

	switch id + "/" + codec {
	case "audio/opus":
		mime = webrtc.MimeTypeOpus
	case "video/h264":
		mime = webrtc.MimeTypeH264
	case "video/vpx", "video/vp8":
		mime = webrtc.MimeTypeVP8
	case "video/vp9":
		mime = webrtc.MimeTypeVP9
	default:
		return nil, fmt.Errorf("unsupported codec %s:%s", id, codec)
	}

	return webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mime}, id, label)
}

func (p *Peer) handleICEState(state webrtc.ICEConnectionState) {
	p.log.Debug().Str("state", state.String()).Msg("rtc [ice] connection state change")
	switch state {
	case webrtc.ICEConnectionStateChecking:
		// nothing
	case webrtc.ICEConnectionStateConnected:
	case webrtc.ICEConnectionStateFailed:
		p.log.Error().Msgf("WebRTC connection fail! connection: %v, ice: %v, gathering: %v, signalling: %v",
			p.c.ConnectionState(), p.c.ICEConnectionState(), p.c.ICEGatheringState(),
			p.c.SignalingState())
		// make ICE restart
	case webrtc.ICEConnectionStateDisconnected:
	case webrtc.ICEConnectionStateClosed:
		p.Disconnect()
	default:
		p.log.Debug().Msg("ICE state is not handled!")
	}
}

func (p *Peer) Channel(label string, conf *webrtc.DataChannelInit, onMessage func([]byte)) (*webrtc.DataChannel, error) {
	config := cmp.Or(conf, &DefaultChannelParams)

	ch, err := p.c.CreateDataChannel(label, config)
	if err != nil {
		return nil, err
	}

	ch.OnOpen(func() { p.log.Debug().Uint16("id", *ch.ID()).Msgf("rtc [chan] [%v] opened", ch.Label()) })
	ch.OnMessage(func(m webrtc.DataChannelMessage) { onMessage(m.Data) })
	ch.OnClose(func() { p.log.Debug().Msgf("rtc [chan] [%v] closed", ch.Label()) })
	ch.OnError(p.logx)

	p.log.Debug().Msgf("rtc [chan] [%v] added", label)

	return ch, nil
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

func (p *Peer) Disconnect() {
	if p.c == nil {
		return
	}
	if p.c.ConnectionState() < webrtc.PeerConnectionStateDisconnected {
		// ignore this due to DTLS fatal: conn is closed
		_ = p.c.Close()
	}
	p.log.Debug().Msg("rtc stopped")
}

func (p *Peer) InitMedia(vCodec string, aCodec string) (err error) {
	if p.v, err = newTrack("video", "video", vCodec); err != nil {
		return err
	}

	if p.a, err = newTrack("audio", "audio", aCodec); err != nil {
		return err
	}

	sendOnly := webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}

	if _, err = p.c.AddTransceiverFromTrack(p.v, sendOnly); err != nil {
		return err
	}
	if _, err = p.c.AddTransceiverFromTrack(p.a, sendOnly); err != nil {
		return err
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	for _, rtpSender := range p.c.GetSenders() {
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()
	}

	return nil
}

func (p *Peer) OnMessage(fn func(data []byte)) {
	p.onMessage = fn
}

func (p *Peer) logx(err error) { p.log.Error().Err(err) }

func fromJson[T any](data string) (T, error) {
	x := new(T)
	return *x, json.Unmarshal([]byte(data), x)
}

func toJson[T any](data T) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
