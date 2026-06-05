package webrtc

import (
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
}

var samplePool sync.Pool

type Decoder func(data string, obj any) error

func New(log *logger.Logger, api *ApiFactory) *Peer { return &Peer{api: api, log: log} }

func (p *Peer) NewConnection(vCodec, aCodec string, onICECandidate func(ice any)) (err error) {
	if p.conn != nil && p.conn.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return
	}
	p.log.Debug().Msg("WebRTC start")
	if p.conn, err = p.api.NewPeer(); err != nil {
		return
	}
	p.conn.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		p.log.Debug().Msgf("WebRTC state change: %v", pcs)
		if pcs == webrtc.PeerConnectionStateConnected {
			p.log.Info().Msg("Connected")
		}
	})
	p.conn.OnICECandidate(p.handleICECandidate(onICECandidate))

	// plug in the [video] track (out)
	if p.v, err = p.AddTrack("video", "video", vCodec); err != nil {
		return err
	}

	// plug in the [audio] track (out)
	if p.a, err = p.AddTrack("audio", "audio", aCodec); err != nil {
		return err
	}

	p.conn.OnICEConnectionStateChange(p.handleICEState(func() {}))

	p.conn.OnDataChannel(func(ch *webrtc.DataChannel) {
		p.log.Debug().Msgf("Added [%s] datachannel", ch.Label())

		if ch.Label() == "data" {
			p.d = ch
			ch.OnMessage(func(m webrtc.DataChannelMessage) {
				if len(m.Data) == 0 || p.OnMessage == nil {
					return
				}
				p.OnMessage(m.Data)
			})
			ch.OnOpen(func() {
				p.log.Debug().Uint16("id", *ch.ID()).Msgf("Data channel [%v] opened", ch.Label())
			})
			ch.OnError(p.logx)
			ch.OnClose(func() { p.log.Debug().Msgf("Data channel [%v] has been closed", ch.Label()) })
		}
	})

	return nil
}

func (p *Peer) AddTrack(id, label, codec string) (*webrtc.TrackLocalStaticSample, error) {
	track, err := newTrack(id, label, codec)
	if err != nil {
		return nil, err
	}
	as, err := p.conn.AddTrack(track)
	if err != nil {
		return nil, err
	}
	// Read incoming RTCP packets
	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := as.Read(buf); err != nil {
				return
			}
		}
	}()
	p.log.Debug().Msgf("Added [%s] track", track.Codec().MimeType)
	return track, nil
}

func (p *Peer) AddDataChannel() error {
	return p.AddChannel("data", func(data []byte) {
		if len(data) == 0 || p.OnMessage == nil {
			return
		}
		p.OnMessage(data)
	})
}

func (p *Peer) OfferAnswer(offer bool) (*webrtc.SessionDescription, error) {
	opts := webrtc.OfferAnswerOptions{ICETricklingSupported: true}

	var sdp webrtc.SessionDescription
	var err error

	if offer {
		sdp, err = p.conn.CreateOffer(&webrtc.OfferOptions{OfferAnswerOptions: opts})
	} else {
		sdp, err = p.conn.CreateAnswer(&webrtc.AnswerOptions{OfferAnswerOptions: opts})
	}
	if err != nil {
		return nil, err
	}
	p.log.Debug().Str("type", sdp.Type.String()).Msg("SDP")

	if err = p.conn.SetLocalDescription(sdp); err != nil {
		return nil, err
	}

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
		case "vp9":
			mime = webrtc.MimeTypeVP9
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

func (p *Peer) AddCandidate(candidate string, decoder Decoder) error {
	// !to add test when the connection is closed but it is still
	// receiving ice candidates

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

func (p *Peer) AddChannel(label string, onMessage func([]byte)) error {
	ch, err := p.addDataChannel(label)
	if err != nil {
		return err
	}
	if label == "data" {
		p.d = ch
	}
	ch.OnMessage(func(m webrtc.DataChannelMessage) { onMessage(m.Data) })
	p.log.Debug().Msgf("Added [%v] chan", label)
	return nil
}

func (p *Peer) Disconnect() {
	if p.conn == nil {
		return
	}
	if p.conn.ConnectionState() < webrtc.PeerConnectionStateDisconnected {
		// ignore this due to DTLS fatal: conn is closed
		_ = p.conn.Close()
	}
	p.log.Debug().Msg("WebRTC stop")
}

// addDataChannel creates new WebRTC data channel.
// Default params -- ordered: true, negotiated: false.
func (p *Peer) addDataChannel(label string) (*webrtc.DataChannel, error) {
	ch, err := p.conn.CreateDataChannel(label, &webrtc.DataChannelInit{
		Ordered:        new(bool),
		MaxRetransmits: new(uint16),
	})
	if err != nil {
		return nil, err
	}
	ch.OnOpen(func() {
		p.log.Debug().Uint16("id", *ch.ID()).Msgf("Data channel [%v] opened", ch.Label())
	})
	ch.OnError(p.logx)
	ch.OnClose(func() { p.log.Debug().Msgf("Data channel [%v] has been closed", ch.Label()) })
	return ch, nil
}

func (p *Peer) logx(err error) { p.log.Error().Err(err) }
