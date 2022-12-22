package webrtc

import (
	"fmt"
	"strings"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type WebRTC struct {
	api         *ApiFactory
	conf        conf.Webrtc
	conn        *webrtc.PeerConnection
	isConnected bool
	log         *logger.Logger
	OnMessage   func(data []byte)

	aTrack *webrtc.TrackLocalStaticSample
	vTrack *webrtc.TrackLocalStaticSample
	dTrack *webrtc.DataChannel
}

type Decoder func(data string, obj interface{}) error

func NewWebRTC(conf conf.Webrtc, log *logger.Logger, api *ApiFactory) *WebRTC {
	return &WebRTC{api: api, conf: conf, log: log}
}

func (w *WebRTC) NewCall(vCodec, aCodec string, onICECandidate func(ice interface{})) (sdp interface{}, err error) {
	if w.isConnected {
		w.log.Warn().Msg("Strange multiple init connection calls with the same peer")
		return
	}
	w.log.Info().Msg("WebRTC start")
	if w.conn, err = w.api.NewPeer(); err != nil {
		return "", err
	}
	w.conn.OnICECandidate(w.handleICECandidate(onICECandidate))
	// plug in the [video] track (out)
	video, err := newTrack("video", "game-video", vCodec)
	if err != nil {
		return "", err
	}
	if _, err = w.conn.AddTrack(video); err != nil {
		return "", err
	}
	w.vTrack = video
	w.log.Debug().Msgf("Added [%s] track", video.Codec().MimeType)

	// plug in the [audio] track (out)
	audio, err := newTrack("audio", "game-audio", aCodec)
	if err != nil {
		return "", err
	}
	if _, err = w.conn.AddTrack(audio); err != nil {
		return "", err
	}
	w.log.Debug().Msgf("Added [%s] track", audio.Codec().MimeType)
	w.aTrack = audio

	// plug in the [input] data channel (in)
	if err = w.addInputChannel("game-input"); err != nil {
		return "", err
	}
	w.log.Debug().Msg("Added input channel ")

	w.conn.OnICEConnectionStateChange(w.handleICEState(func() {
		w.log.Info().Msg("Start streaming")
	}))
	// Stream provider supposes to send offer
	offer, err := w.conn.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	w.log.Info().Msg("Created Offer")

	err = w.conn.SetLocalDescription(offer)
	if err != nil {
		return "", err
	}

	return offer, nil
}

func (w *WebRTC) SetRemoteSDP(sdp string, decoder Decoder) error {
	var answer webrtc.SessionDescription
	if err := decoder(sdp, &answer); err != nil {
		return err
	}
	if err := w.conn.SetRemoteDescription(answer); err != nil {
		w.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}
	w.log.Debug().Msg("Set Remote Description")
	return nil
}

func (w *WebRTC) WriteVideo(sample media.Sample) error { return w.vTrack.WriteSample(sample) }

func (w *WebRTC) WriteAudio(sample media.Sample) error { return w.aTrack.WriteSample(sample) }

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
		case "vp8":
			mime = webrtc.MimeTypeVP8
		}
	}
	if mime == "" {
		return nil, fmt.Errorf("unsupported codec %s:%s", id, codec)
	}
	return webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mime}, id, label)
}

func (w *WebRTC) handleICECandidate(callback func(interface{})) func(*webrtc.ICECandidate) {
	return func(ice *webrtc.ICECandidate) {
		// ICE gathering finish condition
		if ice == nil {
			callback(nil)
			w.log.Debug().Msg("ICE gathering was complete probably")
			return
		}
		candidate := ice.ToJSON()
		w.log.Debug().Str("candidate", candidate.Candidate).Msg("ICE")
		callback(&candidate)
	}
}

func (w *WebRTC) handleICEState(onConnect func()) func(webrtc.ICEConnectionState) {
	return func(state webrtc.ICEConnectionState) {
		w.log.Debug().Str(".state", state.String()).Msg("ICE")
		switch state {
		case webrtc.ICEConnectionStateChecking:
			// nothing
		case webrtc.ICEConnectionStateConnected:
			w.isConnected = true
			onConnect()
		case webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed,
			webrtc.ICEConnectionStateDisconnected:
			w.Disconnect()
		default:
			w.log.Debug().Msg("ICE state is not handled!")
		}
	}
}

func (w *WebRTC) AddCandidate(candidate string, decoder Decoder) error {
	var iceCandidate webrtc.ICECandidateInit
	if err := decoder(candidate, &iceCandidate); err != nil {
		return err
	}
	if err := w.conn.AddICECandidate(iceCandidate); err != nil {
		return err
	}
	w.log.Debug().Str("candidate", iceCandidate.Candidate).Msg("Ice")
	return nil
}

func (w *WebRTC) Disconnect() {
	if !w.isConnected {
		return
	}
	w.isConnected = false
	if w.conn != nil {
		if err := w.conn.Close(); err != nil {
			w.log.Error().Err(err).Msg("WebRTC close")
		}
		w.conn = nil
	}
	w.log.Info().Msg("WebRTC stop")
}

func (w *WebRTC) IsConnected() bool { return w.isConnected }

func (w *WebRTC) SendMessage(data []byte) { _ = w.dTrack.Send(data) }

// addInputChannel creates a new WebRTC data channel for user input.
// Default params -- ordered: true, negotiated: false.
func (w *WebRTC) addInputChannel(label string) error {
	ch, err := w.conn.CreateDataChannel(label, nil)
	if err != nil {
		return err
	}
	ch.OnOpen(func() {
		w.log.Debug().Str("label", ch.Label()).Uint16("id", *ch.ID()).Msg("Data channel [input] opened")
	})
	ch.OnError(w.logx)
	ch.OnMessage(func(mess webrtc.DataChannelMessage) {
		// echo string messages (e.g. ping/pong)
		if mess.IsString {
			w.logx(ch.Send(mess.Data))
			return
		}
		w.OnMessage(mess.Data)
	})
	w.dTrack = ch
	ch.OnClose(func() { w.log.Debug().Msg("Data channel [input] has been closed") })
	return nil
}

func (w *WebRTC) logx(err error) { w.log.Error().Err(err) }
