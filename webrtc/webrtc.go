// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go
package webrtc

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/giongto35/cloud-game/config"
	vpxEncoder "github.com/giongto35/cloud-game/vpx-encoder"
	"github.com/pion/webrtc"
	"github.com/pion/webrtc/pkg/media"
	uuid "github.com/satori/go.uuid"
)

var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		panic(err)
	}
	err = gz.Flush()
	if err != nil {
		panic(err)
	}
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		panic(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		panic(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return res
}

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

		ImageChannel: make(chan []byte, 2),
		AudioChannel: make(chan []byte, 1000),
		InputChannel: make(chan int, 2),
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

	connection  *webrtc.PeerConnection
	encoder     *vpxEncoder.VpxEncoder
	isConnected bool
	isClosed    bool
	// for yuvI420 image
	ImageChannel chan []byte
	AudioChannel chan []byte
	InputChannel chan int

	Done     chan struct{}
	lastTime time.Time
	curFPS   int

	RoomID string
}

// StartClient start webrtc
func (w *WebRTC) StartClient(remoteSession string, width, height int) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			w.StopClient()
		}
	}()

	// reset client
	if w.isConnected {
		w.StopClient()
		time.Sleep(2 * time.Second)
	}

	encoder, err := vpxEncoder.NewVpxEncoder(width, height, 20, 1200, 5)
	if err != nil {
		return "", err
	}
	w.encoder = encoder

	log.Println("=== StartClient ===")

	w.connection, err = webrtc.NewPeerConnection(webrtcconfig)
	if err != nil {
		return "", err
	}

	vp8Track, err := w.connection.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "pion2")
	if err != nil {
		return "", err
	}
	_, err = w.connection.AddTrack(vp8Track)
	if err != nil {
		return "", err
	}

	// audio track
	dfalse := false
	dtrue := true
	var d0 uint16 = 0
	var d1 uint16 = 1
	audioTrack, err := w.connection.CreateDataChannel("b", &webrtc.DataChannelInit{
		Ordered:        &dfalse,
		MaxRetransmits: &d0,
		Negotiated:     &dtrue,
		ID:             &d1,
	})
	if err != nil {
		return "", err
	}

	// input channel
	inputTrack, err := w.connection.CreateDataChannel("a", &webrtc.DataChannelInit{
		Ordered:    &dtrue,
		Negotiated: &dtrue,
		ID:         &d0,
	})

	inputTrack.OnOpen(func() {
		log.Printf("Data channel '%s'-'%d' open.\n", inputTrack.Label(), inputTrack.ID())
	})

	// Register text message handling
	inputTrack.OnMessage(func(msg webrtc.DataChannelMessage) {
		//layout .:= "2006-01-02T15:04:05.000Z"
		//if t, err := time.Parse(layout, string(msg.Data[1])); err == nil {
		//fmt.Println("Delay ", time.Now().Sub(t))
		//} else {
		w.InputChannel <- int(msg.Data[0])
		//}
	})

	inputTrack.OnClose(func() {
		fmt.Println("closed webrtc")
		w.Done <- struct{}{}
		close(w.Done)
	})

	// WebRTC state callback
	w.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			go func() {
				w.isConnected = true
				log.Println("ConnectionStateConnected")
				w.startStreaming(vp8Track, audioTrack)
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

	answer, err := w.connection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

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
	log.Println("===StopClient===")
	w.isConnected = false
	if w.encoder != nil {
		w.encoder.Release()
	}
	if w.connection != nil {
		w.connection.Close()
	}
	w.connection = nil
}

// IsConnected comment
func (w *WebRTC) IsConnected() bool {
	return w.isConnected
}

// func (w *WebRTC) startStreaming(vp8Track *webrtc.Track, opusTrack *webrtc.Track) {
func (w *WebRTC) startStreaming(vp8Track *webrtc.Track, audioTrack *webrtc.DataChannel) {
	log.Println("Start streaming")
	// send screenshot
	go func() {
		for w.isConnected {
			yuv := <-w.ImageChannel
			if len(w.encoder.Input) < cap(w.encoder.Input) {
				w.encoder.Input <- yuv
			}
		}
	}()

	// receive frame buffer
	go func() {
		for w.isConnected {
			bs := <-w.encoder.Output
			if *config.IsMonitor {
				log.Println("FPS : ", w.calculateFPS())
			}
			vp8Track.WriteSample(media.Sample{Data: bs, Samples: 1})
		}
	}()

	// send audio
	go func() {
		for w.isConnected {
			data := <-w.AudioChannel
			// time.Sleep()
			// time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			audioTrack.Send(data)
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
