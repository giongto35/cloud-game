// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go
package webrtc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/giongto35/cloud-game/config"
	"github.com/gofrs/uuid"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

// NewWebRTC create
func NewWebRTC() *WebRTC {
	w := &WebRTC{
		ID: uuid.Must(uuid.NewV4()).String(),

		ImageChannel: make(chan []byte, 30),
		AudioChannel: make(chan []byte, 30),
		InputChannel: make(chan int, 100),
	}
	return w
}

type InputDataPair struct {
	data int
	time time.Time
}

// WebRTC connection
type WebRTC struct {
	ID string

	connection *webrtc.PeerConnection

	isConnected bool
	isClosed    bool
	// for yuvI420 image
	ImageChannel chan []byte
	AudioChannel chan []byte
	InputChannel chan int

	Done     bool
	lastTime time.Time
	curFPS   int

	RoomID string
}

// StartClient start webrtc
func (w *WebRTC) StartClient(remoteSession string, iceCandidates []string) (string, error) {
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

	if config.Codec == config.CODEC_H264 {
		videoTrack, err = w.connection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", "pion2")
	} else {
		videoTrack, err = w.connection.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "pion2")
	}
	if err != nil {
		return "", err
	}
	_, err = w.connection.AddTrack(videoTrack)
	if err != nil {
		return "", err
	}

	// audio track
	opusTrack, err := w.connection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion2b")
	if err != nil {
		return "", err
	}
	_, err = w.connection.AddTrack(opusTrack)
	if err != nil {
		return "", err
	}

	dfalse := false
	dtrue := true
	var d0 uint16 = 0

	// input channel
	inputTrack, err := w.connection.CreateDataChannel("a", &webrtc.DataChannelInit{
		Ordered:    &dfalse,
		Negotiated: &dtrue,
		ID:         &d0,
	})

	inputTrack.OnOpen(func() {
		log.Printf("Data channel '%s'-'%d' open.\n", inputTrack.Label(), inputTrack.ID())
	})

	// Register text message handling
	inputTrack.OnMessage(func(msg webrtc.DataChannelMessage) {
		// TODO: Can add recover here + generalize
		w.InputChannel <- int(msg.Data[1])<<8 + int(msg.Data[0])
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

	// TODO: take a look at this
	w.connection.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		log.Println(iceCandidate)
	})

	offer := webrtc.SessionDescription{}

	Decode(remoteSession, &offer)

	err = w.connection.SetRemoteDescription(offer)
	if err != nil {
		return "", err
	}

	// Parse candidates list
	// This logic is wrong
	for _, bcandidate := range iceCandidates {
		iceCandidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(bcandidate), &iceCandidate); err != nil {
			log.Println("Cannot parse ", bcandidate)
			continue
		}
		log.Println("Add iceCandidate: ", iceCandidate)
		w.connection.AddICECandidate(iceCandidate)
	}

	answer, err := w.connection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	err = w.connection.SetLocalDescription(answer)
	if err != nil {
		return "", err
	}

	// Sendback answer from server
	localSession := Encode(answer)
	return localSession, nil
}

func (w *WebRTC) AttachRoomID(roomID string) {
	w.RoomID = roomID
}

// TODO: Take a look at this
func (w *WebRTC) AddCandidate(candidate webrtc.ICECandidateInit) {
	err := w.connection.AddICECandidate(candidate)
	if err != nil {
		log.Println("Cannot add candidate: ", err)
	}
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
}

// IsConnected comment
func (w *WebRTC) IsConnected() bool {
	return w.isConnected
}

// func (w *WebRTC) startStreaming(vp8Track *webrtc.Track, opusTrack *webrtc.Track) {
func (w *WebRTC) startStreaming(vp8Track *webrtc.Track, opusTrack *webrtc.Track) {
	log.Println("Start streaming")
	// receive frame buffer
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from err", r)
			}
		}()

		for data := range w.ImageChannel {
			if *config.IsMonitor {
				log.Println("Encoding FPS : ", w.calculateFPS())
			}
			err := vp8Track.WriteSample(media.Sample{Data: data, Samples: 1})
			if err != nil {
				log.Println("Warn: Err write sample: ", err)
			}
		}
	}()

	// send audio
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from err", r)
			}
		}()

		for data := range w.AudioChannel {
			if !w.isConnected {
				return
			}
			err := opusTrack.WriteSample(media.Sample{Data: data, Samples: uint32(len(data))})
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
