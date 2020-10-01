// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go
package webrtc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

// TODO: double check if no need TURN server here
var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

type InputDataPair struct {
	data int
	time time.Time
}

type WebFrame struct {
	Data      []byte
	Timestamp uint32
}

// WebRTC connection
type WebRTC struct {
	ID string

	connection  *webrtc.PeerConnection
	isConnected bool
	isClosed    bool
	// for yuvI420 image
	ImageChannel    chan WebFrame
	AudioChannel    chan []byte
	VoiceInChannel  chan []byte
	VoiceOutChannel chan []byte
	InputChannel    chan []byte

	Done     bool
	lastTime time.Time
	curFPS   int

	RoomID string

	// store thing related to game
	GameMeta GameMeta
}

// Game Meta
type GameMeta struct {
	PlayerIndex int
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
func NewWebRTC() *WebRTC {
	w := &WebRTC{
		ID: uuid.Must(uuid.NewV4()).String(),

		ImageChannel:    make(chan WebFrame, 30),
		AudioChannel:    make(chan []byte, 1),
		VoiceInChannel:  make(chan []byte, 1),
		VoiceOutChannel: make(chan []byte, 1),
		InputChannel:    make(chan []byte, 100),
	}
	return w
}

// StartClient start webrtc
func (w *WebRTC) StartClient(isMobile bool, iceCB OnIceCallback) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			w.StopClient()
		}
	}()
	var err error
	var videoTrack *webrtc.Track

	// reset client
	if w.isConnected {
		w.StopClient()
		time.Sleep(2 * time.Second)
	}

	log.Println("=== StartClient ===")
	w.connection, err = webrtc.NewPeerConnection(webrtcconfig)
	if err != nil {
		return "", err
	}

	// add video track
	if util.GetVideoEncoder(isMobile) == config.CODEC_H264 {
		videoTrack, err = w.connection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", "game-video")
	} else {
		videoTrack, err = w.connection.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "game-video")
	}
	if err != nil {
		return "", err
	}

	_, err = w.connection.AddTrack(videoTrack)
	if err != nil {
		return "", err
	}
	log.Println("Add video track")

	// add audio track
	opusTrack, err := w.connection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "game-audio")
	if err != nil {
		return "", err
	}
	_, err = w.connection.AddTrack(opusTrack)
	if err != nil {
		return "", err
	}

	_, err = w.connection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})

	// create data channel for input, and register callbacks
	// order: true, negotiated: false, id: random
	inputTrack, err := w.connection.CreateDataChannel("game-input", nil)

	inputTrack.OnOpen(func() {
		log.Printf("Data channel '%s'-'%d' open.\n", inputTrack.Label(), inputTrack.ID())
	})

	// Register text message handling
	inputTrack.OnMessage(func(msg webrtc.DataChannelMessage) {
		// TODO: Can add recover here
		w.InputChannel <- msg.Data
	})

	inputTrack.OnClose(func() {
		log.Println("Data channel closed")
		log.Println("Closed webrtc")
	})

	// WebRTC state callback
	w.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			go func() {
				w.isConnected = true
				log.Println("ConnectionStateConnected")
				w.startStreaming(videoTrack, opusTrack)
			}()

		}
		if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateDisconnected {
			w.StopClient()
		}
	})

	w.connection.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate != nil {
			log.Println("OnIceCandidate:", iceCandidate.ToJSON().Candidate)
			candidate, err := Encode(iceCandidate.ToJSON())
			if err != nil {
				log.Println("Encode IceCandidate failed: " + iceCandidate.ToJSON().Candidate)
				return
			}
			iceCB(candidate)
		} else {
			// finish, send null
			iceCB("")
		}

	})

	w.connection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		//NOTE: High CPU due to constantly for loop. Turn it off first, Fix it later.
		//rtpBuf := make([]byte, 1400)

		//log.Println("Received Voice from Client")
		//for {
		//if w.RoomID == "" {
		//// skip sending voice when game is not running
		//continue
		//}

		//i, err := remoteTrack.Read(rtpBuf)
		//// TODO: can receive track but the voice doesn't work
		//if err == nil {
		//w.VoiceInChannel <- rtpBuf[:i]
		//}
		//}

	})

	// Stream provider supposes to send offer
	offer, err := w.connection.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	log.Println("Created Offer")

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

func (w *WebRTC) AttachRoomID(roomID string) {
	w.RoomID = roomID
}

func (w *WebRTC) SetRemoteSDP(remoteSDP string) error {
	var answer webrtc.SessionDescription
	err := Decode(remoteSDP, &answer)
	if err != nil {
		log.Println("Decode remote sdp from peer failed")
		return err
	}

	err = w.connection.SetRemoteDescription(answer)
	if err != nil {
		log.Println("Set remote description from peer failed")
		return err
	}

	log.Println("Set Remote Description")
	return nil
}

func (w *WebRTC) AddCandidate(candidate string) error {
	var iceCandidate webrtc.ICECandidateInit
	err := Decode(candidate, &iceCandidate)
	if err != nil {
		log.Println("Decode Ice candidate from peer failed")
		return err
	}
	log.Println("Decoded Ice: " + iceCandidate.Candidate)

	err = w.connection.AddICECandidate(iceCandidate)
	if err != nil {
		log.Println("Add Ice candidate from peer failed")
		return err
	}

	log.Println("Add Ice Candidate: " + iceCandidate.Candidate)
	return nil
}

// StopClient disconnect
func (w *WebRTC) StopClient() {
	// if stopped, bypass
	if w.isConnected == false {
		return
	}

	log.Println("===StopClient===")
	w.isConnected = false
	if w.connection != nil {
		w.connection.Close()
	}
	w.connection = nil
	//close(w.InputChannel)
	// webrtc is producer, so we close
	// NOTE: ImageChannel is waiting for input. Close in writer is not correct for this
	close(w.ImageChannel)
	close(w.AudioChannel)
	close(w.VoiceInChannel)
	close(w.VoiceOutChannel)
}

// IsConnected comment
func (w *WebRTC) IsConnected() bool {
	return w.isConnected
}

func (w *WebRTC) startStreaming(vp8Track *webrtc.Track, opusTrack *webrtc.Track) {
	log.Println("Start streaming")
	// receive frame buffer
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from err", r)
				log.Println(debug.Stack())
			}
		}()

		for data := range w.ImageChannel {
			packets := vp8Track.Packetizer().Packetize(data.Data, 1)
			for _, p := range packets {
				p.Header.Timestamp = data.Timestamp
				err := vp8Track.WriteRTP(p)
				if err != nil {
					log.Println("Warn: Err write sample: ", err)
					break
				}
			}
		}
	}()

	// send audio
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from err", r)
				log.Println(debug.Stack())
			}
		}()

		for data := range w.AudioChannel {
			if !w.isConnected {
				return
			}
			err := opusTrack.WriteSample(media.Sample{
				Data:    data,
				Samples: uint32(config.AUDIO_FRAME / config.AUDIO_CHANNELS),
			})
			if err != nil {
				log.Println("Warn: Err write sample: ", err)
			}
		}
	}()

	// send voice
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from err", r)
				log.Println(debug.Stack())
			}
		}()

		for data := range w.VoiceOutChannel {
			if !w.isConnected {
				return
			}
			_, err := opusTrack.Write(data)
			if err != nil {
				log.Println("Warn: Err write sample: ", err)
			}
		}
	}()
}

func (w *WebRTC) calculateFPS() int {
	elapsedTime := time.Now().Sub(w.lastTime)
	w.lastTime = time.Now()
	curFPS := time.Second / elapsedTime
	w.curFPS = int(float32(w.curFPS)*0.9 + float32(curFPS)*0.1)
	return w.curFPS
}
