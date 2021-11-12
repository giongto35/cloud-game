package webrtc

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type (
	AudioFrame struct {
		Data     []byte
		Duration time.Duration
	}
	VideoFrame struct {
		Data      []byte
		Timestamp uint32
	}
)

type WebRTC struct {
	id                        network.Uid
	conf                      conf.Webrtc
	connection                *webrtc.PeerConnection
	connectionBase            *Peer
	globalVideoFrameTimestamp uint32
	isConnected               bool
	ImageChannel              chan VideoFrame
	AudioChannel              chan AudioFrame
	InputChannel              chan []byte
	RoomID                    string
	PlayerIndex               int
	done                      chan struct{}
	log                       *logger.Logger
}

type Decoder func(data string, obj interface{}) error

func NewWebRTC(conf conf.Webrtc, log *logger.Logger) (*WebRTC, error) {
	w := &WebRTC{
		id:           network.NewUid(),
		ImageChannel: make(chan VideoFrame, 30),
		AudioChannel: make(chan AudioFrame, 1),
		InputChannel: make(chan []byte, 100),
		done:         make(chan struct{}, 1),
		conf:         conf,
		log:          log,
	}
	conn, err := DefaultPeerConnection(conf, &w.globalVideoFrameTimestamp, log)
	if err != nil {
		return nil, err
	}
	w.connectionBase = conn
	return w, nil
}

func (w *WebRTC) NewCall(vCodec, aCodec string, onICECandidate func(ice interface{})) (sdp interface{}, err error) {
	if w.isConnected {
		w.log.Warn().Msg("Strange multiple init connection calls with the same peer")
		return
	}
	w.log.Info().Str("id", w.id.Short()).Msgf("WebRTC start (uid:%s)", w.id)
	if w.connection, err = w.connectionBase.NewPeer(); err != nil {
		return "", err
	}
	w.connection.OnICECandidate(w.handleICECandidate(onICECandidate))
	// plug in the [video] track (out)
	video, err := newTrack("video", "game-video", vCodec)
	if err != nil {
		return "", err
	}
	if _, err = w.connection.AddTrack(video); err != nil {
		return "", err
	}
	w.log.Debug().Msgf("Added [%s] track", video.Codec().MimeType)

	// plug in the [audio] track (out)
	audio, err := newTrack("audio", "game-audio", aCodec)
	if err != nil {
		return "", err
	}
	if _, err = w.connection.AddTrack(audio); err != nil {
		return "", err
	}
	w.log.Debug().Msgf("Added [%s] track", audio.Codec().MimeType)

	// plug in the [input] data channel (in)
	if err = w.addInputChannel("game-input"); err != nil {
		return "", err
	}
	w.log.Debug().Msg("Added input channel ")

	w.connection.OnICEConnectionStateChange(w.handleICEState(func() { go w.Stream(video, audio) }))
	// Stream provider supposes to send offer
	offer, err := w.connection.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	w.log.Info().Msg("Created Offer")

	err = w.connection.SetLocalDescription(offer)
	if err != nil {
		return "", err
	}

	return offer, nil
}

// SetRoom sets room identifier for the current WebRTC connection.
func (w *WebRTC) SetRoom(id string) { w.RoomID = id }

func (w *WebRTC) SetRemoteSDP(sdp string, decoder Decoder) error {
	var answer webrtc.SessionDescription
	if err := decoder(sdp, &answer); err != nil {
		return err
	}
	if err := w.connection.SetRemoteDescription(answer); err != nil {
		w.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}
	w.log.Debug().Msg("Set Remote Description")
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
		w.log.Debug().Str("id", w.id.Short()).Str(".state", state.String()).Msg("ICE")
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

func (w *WebRTC) GetId() string { return w.id.String() }

func (w *WebRTC) AddCandidate(candidate string, decoder Decoder) error {
	var iceCandidate webrtc.ICECandidateInit
	if err := decoder(candidate, &iceCandidate); err != nil {
		return err
	}
	if err := w.connection.AddICECandidate(iceCandidate); err != nil {
		return err
	}
	w.log.Debug().Str("candidate", iceCandidate.Candidate).Msg("Ice")
	return nil
}

func (w *WebRTC) Disconnect() {
	if !w.IsConnected() {
		return
	}
	w.isConnected = false
	if w.connection != nil {
		if err := w.connection.Close(); err != nil {
			w.log.Error().Err(err).Msg("WebRTC close")
		}
	}
	w.connection = nil
	close(w.ImageChannel)
	close(w.AudioChannel)
	close(w.done)
	w.log.Info().Msg("WebRTC stop")
}

func (w *WebRTC) IsConnected() bool { return w.isConnected }

func (w *WebRTC) Stream(videoTrack, audioTrack *webrtc.TrackLocalStaticSample) {
	defer func() {
		w.log.Info().Msg("Stop streaming")
		if err := recover(); err != nil {
			w.log.Error().Err(fmt.Errorf("%v", err)).Msg("WebRTC stream crashed")
		}
	}()
	w.log.Info().Msg("Start streaming")
	for {
		select {
		case <-w.done:
			return
		case image := <-w.ImageChannel:
			atomic.StoreUint32(&w.globalVideoFrameTimestamp, image.Timestamp)
			if err := videoTrack.WriteSample(media.Sample{Data: image.Data}); err != nil {
				w.log.Error().Err(err).Msg("Audio sample error")
				return
			}
		case audio := <-w.AudioChannel:
			if !w.IsConnected() {
				return
			}
			err := audioTrack.WriteSample(media.Sample{Data: audio.Data, Duration: audio.Duration})
			if err != nil {
				w.log.Error().Err(err).Msg("Opus sample error")
				return
			}
		}
	}
}

// addInputChannel creates a new WebRTC data channel for user input.
// Default params -- ordered: true, negotiated: false.
func (w *WebRTC) addInputChannel(label string) error {
	ch, err := w.connection.CreateDataChannel(label, nil)
	if err != nil {
		return err
	}
	ch.OnOpen(func() {
		w.log.Debug().Str("label", ch.Label()).Uint16("id", *ch.ID()).Msg("Data channel [input] opened")
	})
	ch.OnError(func(err error) { w.log.Error().Err(err).Msg("Data channel [input]") })
	ch.OnMessage(func(message webrtc.DataChannelMessage) {
		if message.IsString {
			// todo wtf is this magic byte
			_ = ch.Send([]byte{0x42})
			return
		}
		// TODO: Can add recover here
		w.InputChannel <- message.Data
	})
	ch.OnClose(func() { w.log.Debug().Msg("Data channel [input] has been closed") })
	return nil
}
