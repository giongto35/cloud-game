package webrtc

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"

	vpxEncoder "github.com/giongto35/cloud-game/vpx-encoder"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/media"
)

var config = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

func init() {
	//api.mediaEngine.RegisterDefaultCodecs()
	//webrtc.RegisterDefaultCodecs()
}

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
		ImageChannel: make(chan []byte, 2),
		InputChannel: make(chan int, 2),
	}
	return w
}

// WebRTC connection
type WebRTC struct {
	connection  *webrtc.PeerConnection
	encoder     *vpxEncoder.VpxEncoder
	isConnected bool
	isClosed    bool
	// for yuvI420 image
	ImageChannel chan []byte
	InputChannel chan int
}

// StartClient start webrtc
func (w *WebRTC) StartClient(remoteSession string, width, height int) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
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

	fmt.Println("=== StartClient ===")

	w.connection, err = webrtc.NewPeerConnection(config)
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


	// WebRTC state callback
	w.connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			go func() {
				w.isConnected = true
				fmt.Println("ConnectionStateConnected")
				w.startStreaming(vp8Track)
			}()

		}
		if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateDisconnected {
			w.StopClient()
		}
	})

	w.connection.OnICECandidate(func(iceCandidate *webrtc.ICECandidate) {
		fmt.Println(iceCandidate)
	})


	// Data channel callback
	w.connection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			//fmt.Printf("Message from DataChannel '%s': '%s' byte '%b'\n", d.Label(), string(msg.Data), msg.Data)
			i, _ := strconv.Atoi(string(msg.Data))
			w.InputChannel <- i
		})
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

func (w *WebRTC) AddCandidate(candidate webrtc.ICECandidateInit) {
	err := w.connection.AddICECandidate(candidate)
	if err != nil {
		fmt.Println("Cannot add candidate: ", err)
	}
}

// StopClient disconnect
func (w *WebRTC) StopClient() {
	fmt.Println("===StopClient===")
	w.isConnected = false
	if w.encoder != nil {
		w.encoder.Release()
	}
	if w.connection != nil {
		w.connection.Close()
	}
	w.connection = nil
	w.isClosed = true
}

// IsConnected comment
func (w *WebRTC) IsConnected() bool {
	return w.isConnected
}

func (w *WebRTC) IsClosed() bool {
	return w.isClosed
}

func (w *WebRTC) startStreaming(vp8Track *webrtc.Track) {
	fmt.Println("Start streaming")
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
		for i := 0; w.isConnected; i++ {
			bs := <-w.encoder.Output
			vp8Track.WriteSample(media.Sample{Data: bs, Samples: 1})
		}
	}()
}
