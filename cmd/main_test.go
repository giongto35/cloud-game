package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/handler"
	"github.com/giongto35/cloud-game/overlord"
	gamertc "github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc"
)

var host = "http://localhost:8000"

// Test is in cmd, so gamePath is in parent path
var testGamePath = "../games"
var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}}

func initOverlord() *httptest.Server {
	server := overlord.NewServer()
	overlord := httptest.NewServer(http.HandlerFunc(server.WSO))
	return overlord
}

func initServer(t *testing.T, oconn *websocket.Conn) *httptest.Server {
	fmt.Println("Spawn new server")
	handler := handler.NewHandler(oconn, true, testGamePath)
	server := httptest.NewServer(http.HandlerFunc(handler.WS))
	return server
}

func connectTestOverlordServer(t *testing.T, overlordURL string) *websocket.Conn {
	if overlordURL == "" {
		return nil
	} else {
		overlordURL = "ws" + strings.TrimPrefix(overlordURL, "http")
		fmt.Println("connecting to overlord: ", overlordURL)
	}

	oconn, _, err := websocket.DefaultDialer.Dial(overlordURL, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	return oconn
}

func initClient(t *testing.T, host string) (client *cws.Client) {
	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(host, "http")

	// Connect to the server
	fmt.Println("Connecting to server")
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Simulate peerconnection initialization from client
	fmt.Println("Simulating PeerConnection")
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
	log.Println("Browser Client")
	client = cws.NewClient(ws)
	go client.Listen()

	fmt.Println("Sending offer...")
	client.Send(cws.WSPacket{
		ID:   "initwebrtc",
		Data: gamertc.Encode(offer),
	}, nil)
	fmt.Println("Waiting sdp...")

	client.Receive("sdp", func(resp cws.WSPacket) cws.WSPacket {
		fmt.Println("received", resp.Data)
		answer := webrtc.SessionDescription{}
		gamertc.Decode(resp.Data, &answer)
		// Apply the answer as the remote description
		err = peerConnection.SetRemoteDescription(answer)
		if err != nil {
			panic(err)
		}

		return cws.EmptyPacket
	})

	// Request Offer routing
	client.Receive("requestOffer", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Frontend received requestOffer")
		peerConnection, err = webrtc.NewPeerConnection(webrtcconfig)
		if err != nil {
			t.Fatalf("%v", err)
		}

		log.Println("Recreating offer")
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			t.Fatalf("%v", err)
		}

		log.Println("Set localDesc")
		err = peerConnection.SetLocalDescription(offer)
		if err != nil {
			panic(err)
		}
		log.Println("return offer", offer)
		return cws.WSPacket{
			ID:   "initwebrtc",
			Data: gamertc.Encode(offer),
		}
	})

	return client
	// If receive roomID, the server is running correctly
}

func TestSingleServerNoOverlord(t *testing.T) {
	// Init slave server
	s := initServer(t, nil)
	defer s.Close()

	client := initClient(t, s.URL)
	defer client.Close()

	roomID := make(chan string)
	client.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	respRoomID := <-roomID
	if respRoomID == "" {
		fmt.Println("RoomID should not be empty")
		t.Fail()
	}
	fmt.Println("Done")
}

func TestSingleServerOneOverlord(t *testing.T) {
	o := initOverlord()
	defer o.Close()

	oconn := connectTestOverlordServer(t, o.URL)
	defer oconn.Close()
	// Init slave server
	s := initServer(t, oconn)
	defer s.Close()

	client := initClient(t, s.URL)
	defer client.Close()

	fmt.Println("Sending start...")
	roomID := make(chan string)
	client.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	respRoomID := <-roomID
	if respRoomID == "" {
		fmt.Println("RoomID should not be empty")
		t.Fail()
	}
	fmt.Println("Done")
}

func TestTwoServerOneOverlord(t *testing.T) {
	o := initOverlord()
	defer o.Close()

	oconn1 := connectTestOverlordServer(t, o.URL)
	// Init slave server
	s1 := initServer(t, oconn1)
	defer s1.Close()

	oconn2 := connectTestOverlordServer(t, o.URL)
	// TODO: two different oconn
	s2 := initServer(t, oconn2)
	defer s2.Close()

	client1 := initClient(t, s1.URL)
	defer client1.Close()

	roomID := make(chan string)
	client1.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	remoteRoomID := <-roomID
	if remoteRoomID == "" {
		fmt.Println("RoomID should not be empty")
		t.Fail()
	}
	fmt.Println("Done create a room in server 1")

	// ------------------------------------
	// Client2 trying to create a random room and later join the the room on server1
	client2 := initClient(t, s2.URL)
	defer client2.Close()
	// Doing the same create local room.
	localRoomID := make(chan string)
	client2.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		localRoomID <- resp.RoomID
	})

	<-localRoomID

	fmt.Println("Request the room from server 1", remoteRoomID)
	log.Println("Server2 trying to join server1 room")
	// After trying loging in to one session, login to other with the roomID
	client2.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      remoteRoomID,
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	// If receive roomID, the server is running correctly
	fmt.Println("Done")
}

func TestReconnectRoomNoOverlord(t *testing.T) {
	// Init slave server
	s := initServer(t, nil)

	client := initClient(t, s.URL)

	fmt.Println("Sending start...")
	roomID := make(chan string)
	client.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      "",
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	saveRoomID := <-roomID
	if saveRoomID == "" {
		fmt.Println("RoomID should not be empty")
		t.Fail()
	}

	log.Println("Closing room and server")
	client.Close()
	s.Close()
	// Close server and reconnect

	// Respawn slave server
	s = initServer(t, nil)
	defer s.Close()

	client = initClient(t, s.URL)
	defer client.Close()

	fmt.Println("Re-access room ", saveRoomID)
	roomID = make(chan string)
	client.Send(cws.WSPacket{
		ID:          "start",
		Data:        "Contra.nes",
		RoomID:      saveRoomID,
		PlayerIndex: 1,
	}, func(resp cws.WSPacket) {
		fmt.Println("RoomID:", resp.RoomID)
		roomID <- resp.RoomID
	})

	respRoomID := <-roomID
	if respRoomID == "" || respRoomID != saveRoomID {
		fmt.Println("The room ID should be equal to the saved room")
		t.Fail()
	}

	fmt.Println("Done")

}

// This test currently doesn't work
//func TestReconnectRoomWithOverlord(t *testing.T) {
//o := initOverlord()
//defer o.Close()

//oconn := connectTestOverlordServer(t, o.URL)
//defer oconn.Close()
//// Init slave server
//s := initServer(t, oconn)

//client := initClient(t, s.URL)

//fmt.Println("Sending start...")
//roomID := make(chan string)
//client.Send(cws.WSPacket{
//ID:          "start",
//Data:        "Contra.nes",
//RoomID:      "",
//PlayerIndex: 1,
//}, func(resp cws.WSPacket) {
//fmt.Println("RoomID:", resp.RoomID)
//roomID <- resp.RoomID
//})

//saveRoomID := <-roomID
//if saveRoomID == "" {
//fmt.Println("RoomID should not be empty")
//t.Fail()
//}

//log.Println("Closing room and server")
//client.Close()
//s.Close()
//// Close server and reconnect

//log.Println("Server respawn")
//// Init slave server
//s = initServer(t, oconn)
//defer s.Close()

//client = initClient(t, s.URL)
//defer client.Close()

//fmt.Println("Re-access room ", saveRoomID)
//roomID = make(chan string)
//client.Send(cws.WSPacket{
//ID:          "start",
//Data:        "Contra.nes",
//RoomID:      saveRoomID,
//PlayerIndex: 1,
//}, func(resp cws.WSPacket) {
//fmt.Println("RoomID:", resp.RoomID)
//roomID <- resp.RoomID
//})

//respRoomID := <-roomID
//if respRoomID == "" || respRoomID != saveRoomID {
//fmt.Println("The room ID should be equal to the saved room")
//t.Fail()
//}

//fmt.Println("Done")

//}
