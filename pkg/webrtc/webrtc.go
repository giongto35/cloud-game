package webrtc

import (
	"fmt"
	"sync/atomic"
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type WebFrame struct {
	Data      []byte
	Timestamp uint32
}

type WebRTC struct {
	id                        string
	conf                      conf.Config
	connection                *webrtc.PeerConnection
	connectionBase            *Peer
	globalVideoFrameTimestamp uint32
	isConnected               bool
	ImageChannel              chan WebFrame
	AudioChannel              chan []byte
	InputChannel              chan []byte
	Done                      bool
	RoomID                    string
	PlayerIndex               int
	log                       *logger.Logger
}

type Decoder func(data string, obj interface{}) error

func NewWebRTC(conf conf.Config, log *logger.Logger) (*WebRTC, error) {
	w := &WebRTC{
		id:           string(network.NewUid()),
		ImageChannel: make(chan WebFrame, 30),
		AudioChannel: make(chan []byte, 1),
		InputChannel: make(chan []byte, 100),
		conf:         conf,
		log:          log,
	}
	conn, err := DefaultPeerConnection(w.conf.Webrtc, &w.globalVideoFrameTimestamp, log)
	if err != nil {
		return nil, err
	}
	w.connectionBase = conn
	return w, nil
}

func (w *WebRTC) InitConnection(ICECandidateCallback func(ice interface{})) (sdp interface{}, err error) {
	defer func() {
		if err := recover(); err != nil {
			w.log.Error().Err(fmt.Errorf("%v", err)).Msg("WebRTC connection crashed")
			w.StopClient()
		}
	}()

	// reset client
	if w.IsConnected() {
		w.StopClient()
		time.Sleep(2 * time.Second)
	}

	w.log.Info().Msg("Start WebRTC")
	peerConn, err := w.connectionBase.NewPeer()
	if err != nil {
		return "", err
	}
	w.connection = peerConn
	w.connection.OnICECandidate(w.handleICECandidate(ICECandidateCallback))

	// plug in the [video] track (out)
	video, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: w.getVideoCodec()}, "video", "game-video")
	if err != nil {
		return "", err
	}
	if _, err = w.connection.AddTrack(video); err != nil {
		return "", err
	}
	w.log.Debug().Msgf("Added [%s] track", video.Codec().MimeType)

	// plug in the [audio] track (out)
	audio, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "game-audio")
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

	w.connection.OnICEConnectionStateChange(w.handleICEConnectionStateChange(func() { w.startStreaming(video, audio) }))
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
		w.log.Error().Err(err).Msg("SDP decode")
		return err
	}
	if err := w.connection.SetRemoteDescription(answer); err != nil {
		w.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}
	w.log.Debug().Msg("Set Remote Description")
	return nil
}

func (w *WebRTC) handleICECandidate(callback func(interface{})) func(*webrtc.ICECandidate) {
	return func(ice *webrtc.ICECandidate) {
		// ICE gathering finish condition
		if ice == nil {
			callback(nil)
			w.log.Debug().Msg("ICE gathering is complete probably")
			return
		}
		cdt := ice.ToJSON()
		w.log.Debug().Str("candidate", cdt.Candidate).Msg("ICE")
		callback(&cdt)
	}
}

func (w *WebRTC) handleICEConnectionStateChange(connectionCallback func()) func(webrtc.ICEConnectionState) {
	return func(state webrtc.ICEConnectionState) {
		w.log.Debug().Str("state", state.String()).Str("id", w.id).Msg("ICE state")
		switch state {
		case webrtc.ICEConnectionStateConnected:
			w.isConnected = true
			connectionCallback()
		case webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed,
			webrtc.ICEConnectionStateDisconnected:
			w.StopClient()
		default:
			w.log.Debug().Msg("ICE state is not handled!")
		}
	}
}

func (w *WebRTC) GetId() string { return w.id }

func (w *WebRTC) AddCandidate(candidate string, decoder Decoder) error {
	var iceCandidate webrtc.ICECandidateInit
	if err := decoder(candidate, &iceCandidate); err != nil {
		w.log.Error().Err(err).Msg("Ice decode")
		return err
	}
	if err := w.connection.AddICECandidate(iceCandidate); err != nil {
		w.log.Error().Err(err).Msg("Ice pull")
		return err
	}
	w.log.Debug().Str("candidate", iceCandidate.Candidate).Msg("Ice")
	return nil
}

// StopClient disconnect
func (w *WebRTC) StopClient() {
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
	w.log.Info().Msg("WebRTC stop")
}

func (w *WebRTC) IsConnected() bool { return w.isConnected }

func (w *WebRTC) startStreaming(vp8Track *webrtc.TrackLocalStaticSample, opusTrack *webrtc.TrackLocalStaticSample) {
	w.log.Info().Msg("Start streaming")
	// receive frame buffer
	go func() {
		defer func() {
			if err := recover(); err != nil {
				w.log.Error().Err(fmt.Errorf("%v", err)).Msg("WebRTC stream crashed")
			}
		}()

		for data := range w.ImageChannel {
			atomic.StoreUint32(&w.globalVideoFrameTimestamp, data.Timestamp)
			if err := vp8Track.WriteSample(media.Sample{Data: data.Data}); err != nil {
				w.log.Error().Err(err).Msg("Audio sample error")
				break
			}
		}
	}()

	// send audio
	go func() {
		defer func() {
			if err := recover(); err != nil {
				w.log.Error().Err(fmt.Errorf("%v", err)).Msg("WebRTC audio crashed")
			}
		}()

		audioDuration := time.Duration(w.conf.Encoder.Audio.Frame) * time.Millisecond
		for data := range w.AudioChannel {
			if !w.IsConnected() {
				return
			}
			err := opusTrack.WriteSample(media.Sample{Data: data, Duration: audioDuration})
			if err != nil {
				w.log.Error().Err(err).Msg("Opus sample error")
			}
		}
	}()
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
	ch.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			// todo wtf is this magic byte
			_ = ch.Send([]byte{0x42})
			return
		}
		// TODO: Can add recover here
		w.InputChannel <- msg.Data
	})
	ch.OnClose(func() { w.log.Debug().Msg("Data channel [input] has been closed") })
	return nil
}

func (w *WebRTC) getVideoCodec() string {
	switch w.conf.Encoder.Video.Codec {
	case string(encoder.H264):
		return webrtc.MimeTypeH264
	case string(encoder.VPX):
		return webrtc.MimeTypeVP8
	default:
		return webrtc.MimeTypeH264
	}
}
