package webrtc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
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

// WebRTC connection
type WebRTC struct {
	ID string

	connection                *webrtc.PeerConnection
	conf                      webrtcConfig.Config
	globalVideoFrameTimestamp uint32
	defaultConnection         *PeerConnection
	isConnected               bool
	ImageChannel              chan WebFrame
	AudioChannel              chan []byte
	InputChannel              chan []byte

	Done bool

	RoomID      string
	PlayerIndex int
	log         *logger.Logger
}

type OnIceCallback func(candidate string)

// Encode encodes the input in base64
func Encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// Decode decodes the input from base64
func Decode(in string, obj interface{}) error {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}

	return nil
}

// NewWebRTC create
func NewWebRTC(conf webrtcConfig.Config, log *logger.Logger) (*WebRTC, error) {
	w := &WebRTC{
		ID:           string(network.NewUid()),
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
	w.defaultConnection = conn
	return w, nil
}

// StartClient start webrtc
func (w *WebRTC) StartClient(iceCB OnIceCallback) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			w.log.Error().Err(fmt.Errorf("%v", err)).Msg("WebRTC connection crashed")
			w.StopClient()
		}
	}()
	var err error
	var videoTrack *webrtc.TrackLocalStaticSample

	// reset client
	if w.isConnected {
		w.StopClient()
		time.Sleep(2 * time.Second)
	}

	w.log.Info().Msg("Start WebRTC")
	w.connection, err = w.defaultConnection.NewConnection()
	if err != nil {
		return "", err
	}

	// add video track
	rtpCodec := webrtc.RTPCodecCapability{MimeType: w.getVideoCodec()}
	if videoTrack, err = webrtc.NewTrackLocalStaticSample(rtpCodec, "video", "game-video"); err != nil {
		return "", err
	}

	if _, err = w.connection.AddTrack(videoTrack); err != nil {
		return "", err
	}
	w.log.Debug().Msg("Add video track")

	// add audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "game-audio")
	if err != nil {
		return "", err
	}
	_, err = w.connection.AddTrack(audioTrack)
	if err != nil {
		return "", err
	}

	// plug in the [input] data channel
	if err = w.handleInputChannel(); err != nil {
		return "", err
	}

	w.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		w.log.Debug().Str("state", connectionState.String()).Msg("Ice new state")
		if connectionState == webrtc.ICEConnectionStateConnected {
			w.isConnected = true
			w.log.Debug().Str("state", "ConnectionStateConnected").Msg("Ice state")
			w.startStreaming(videoTrack, audioTrack)
		}
		if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateDisconnected {
			w.log.Info().Str("id", w.ID).Str("room", w.RoomID).Msg("WebRTC")
			w.StopClient()
		}
	})

	w.connection.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate != nil {
			cdt := iceCandidate.ToJSON()
			w.log.Debug().Str("candidate", cdt.Candidate).Msg("Ice")
			candidate, err := Encode(cdt)
			if err != nil {
				w.log.Error().Err(err).Str("candidate", cdt.Candidate).Msg("Ice candidate encode fail")
				return
			}
			iceCB(candidate)
		} else {
			// finish, send null
			iceCB("")
		}
	})

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

	localSession, err := Encode(offer)
	if err != nil {
		return "", err
	}

	return localSession, nil
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

func (w *WebRTC) AttachRoomID(roomID string) { w.RoomID = roomID }

func (w *WebRTC) SetRemoteSDP(remoteSDP string) error {
	var answer webrtc.SessionDescription
	err := Decode(remoteSDP, &answer)
	if err != nil {
		w.log.Error().Err(err).Msg("SDP decode")
		return err
	}

	err = w.connection.SetRemoteDescription(answer)
	if err != nil {
		w.log.Error().Err(err).Msg("Set remote description from peer failed")
		return err
	}

	w.log.Debug().Msg("Set Remote Description")
	return nil
}

func (w *WebRTC) AddCandidate(candidate string) error {
	var iceCandidate webrtc.ICECandidateInit
	err := Decode(candidate, &iceCandidate)
	if err != nil {
		w.log.Error().Err(err).Msg("Ice decode")
		return err
	}
	err = w.connection.AddICECandidate(iceCandidate)
	if err != nil {
		w.log.Error().Err(err).Msg("Ice pull")
		return err
	}
	w.log.Debug().Str("candidate", iceCandidate.Candidate).Msg("Ice")
	return nil
}

// StopClient disconnect
func (w *WebRTC) StopClient() {
	// if stopped, bypass
	if !w.isConnected {
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
			if !w.isConnected {
				return
			}
			err := opusTrack.WriteSample(media.Sample{Data: data, Duration: audioDuration})
			if err != nil {
				w.log.Error().Err(err).Msg("Opus sample error")
			}
		}
	}()
}
