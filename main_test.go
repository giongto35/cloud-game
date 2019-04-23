package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gamertc "github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc"
)

var host = "http://localhost:8000"
var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

func initOverlord() *httptest.Server {
	overlord := httptest.NewServer(http.HandlerFunc(wso))
	return overlord
}

func initServer(t *testing.T, overlordURL string) *httptest.Server {
	u := "ws" + strings.TrimPrefix(overlordURL, "http")
	fmt.Println("connecting to overlord: ", u)

	oconn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	oclient = NewOverlordClient(oconn)

	server := httptest.NewServer(http.HandlerFunc(ws))
	return server
}

func initClient(t *testing.T, host string) {
	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(host, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	// Simulate peerconnection initialization from client

	peerConnection, err := webrtc.NewPeerConnection(webrtcconfig)
	if err != nil {
		t.Fatalf("%v", err)
	}

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	// Send offer to server
	client := NewClient(ws)
	go client.listen()

	fmt.Println("Sending offer...")
	client.send(WSPacket{
		ID:   "initwebrtc",
		Data: gamertc.Encode(offer),
	}, nil)
	fmt.Println("Waiting sdp...")

	client.receive("sdp", func(resp WSPacket) WSPacket {
		fmt.Println("received", resp.Data)
		answer := webrtc.SessionDescription{}
		gamertc.Decode(resp.Data, &answer)
		// Apply the answer as the remote description
		err = peerConnection.SetRemoteDescription(answer)
		if err != nil {
			panic(err)
		}

		return EmptyPacket
	})

	time.Sleep(time.Second * 3)
	fmt.Println("Sending start...")

	roomID := make(chan string)
	client.send(WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp WSPacket) {
		fmt.Println("Received response")
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	respRoomID := <-roomID
	if respRoomID == "" {
		fmt.Println("RoomID should not be empty")
		t.Fail()
	}
	fmt.Println("Done")
	// If receive roomID, the server is running correctly
}

//func TestSingleServerNoOverlord(t *testing.T) {
//// Init slave server
//s := initServer(t, "")
//defer s.Close()

//initClient(t, s.URL)
//}

func TestSingleServerOneOverlord(t *testing.T) {
	o := initOverlord()
	defer o.Close()
	// Init slave server
	s := initServer(t, o.URL)
	defer s.Close()

	initClient(t, s.URL)
}
