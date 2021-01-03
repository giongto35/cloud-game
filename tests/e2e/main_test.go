package e2e

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"net/http/httptest"
// 	"os"
// 	"strings"
// 	"testing"
// 	"time"
//
// 	"github.com/giongto35/cloud-game/pkg/cws"
// 	"github.com/giongto35/cloud-game/pkg/overlord"
// 	"github.com/giongto35/cloud-game/pkg/util"
// 	gamertc "github.com/giongto35/cloud-game/pkg/webrtc"
// 	"github.com/giongto35/cloud-game/pkg/worker"
// 	"github.com/gorilla/websocket"
// 	"github.com/pion/webrtc/v2"
// )
//
// var host = "http://localhost:8000"
//
// // Test is in cmd, so gamePath is in parent path
// var testGamePath = "../games"
// var webrtcconfig = webrtc.Configuration{ICEServers: []webrtc.ICEServer{{
// 	URLs: []string{"stun:stun.l.google.com:19302"},
// }}}
//
// func initCoordinator() (*httptest.Server, *httptest.Server) {
// 	server := overlord.NewServer()
// 	overlordWorker := httptest.NewServer(http.HandlerFunc(server.WSO))
// 	overlordBrowser := httptest.NewServer(http.HandlerFunc(server.WS))
// 	return overlordWorker, overlordBrowser
// }
//
// func initWorker(t *testing.T, overlordURL string) *worker.Handler {
// 	fmt.Println("Spawn new worker")
// 	if overlordURL == "" {
// 		return nil
// 	} else {
// 		overlordURL = "ws" + strings.TrimPrefix(overlordURL, "http")
// 		fmt.Println("connecting to overlord: ", overlordURL)
// 	}
//
// 	handler := worker.NewHandler(overlordURL, testGamePath)
//
// 	go handler.Run()
// 	time.Sleep(time.Second)
// 	return handler
// }
//
// func initClient(t *testing.T, host string) (client *cws.Client) {
// 	// Convert http://127.0.0.1 to ws://127.0.0.
// 	u := "ws" + strings.TrimPrefix(host, "http")
//
// 	fmt.Println("Connecting to", u)
// 	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
// 	if err != nil {
// 		t.Fatalf("%v", err)
// 	}
//
// 	handshakedone := make(chan struct{})
// 	// Simulate peerconnection initialization from client
// 	fmt.Println("Simulating PeerConnection")
// 	peerConnection, err := webrtc.NewPeerConnection(webrtcconfig)
// 	if err != nil {
// 		t.Fatalf("%v", err)
// 	}
//
// 	offer, err := peerConnection.CreateOffer(nil)
// 	if err != nil {
// 		t.Fatalf("%v", err)
// 	}
//
// 	// Sets the LocalDescription, and starts our UDP listeners
// 	err = peerConnection.SetLocalDescription(offer)
// 	if err != nil {
// 		panic(err)
// 	}
// 	// Send offer to overlord
// 	log.Println("Browser Client")
// 	client = cws.NewClient(ws)
// 	go client.Listen()
//
// 	fmt.Println("Sending offer...")
// 	client.Send(cws.WSPacket{
// 		ID:   "init_webrtc",
// 		Data: gamertc.Encode(offer),
// 	}, nil)
// 	fmt.Println("Waiting sdp...")
//
// 	client.Receive("sdp", func(resp cws.WSPacket) cws.WSPacket {
// 		log.Println("Received SDP", resp.Data, "client: ", client)
// 		answer := webrtc.SessionDescription{}
// 		gamertc.Decode(resp.Data, &answer)
// 		// Apply the answer as the remote description
// 		err = peerConnection.SetRemoteDescription(answer)
// 		if err != nil {
// 			panic(err)
// 		}
//
// 		// TODO: may block in the second call
// 		handshakedone <- struct{}{}
//
// 		return cws.EmptyPacket
// 	})
//
// 	// Request Offer routing
// 	client.Receive("requestOffer", func(resp cws.WSPacket) cws.WSPacket {
// 		log.Println("Frontend received requestOffer")
// 		peerConnection, err = webrtc.NewPeerConnection(webrtcconfig)
// 		if err != nil {
// 			t.Fatalf("%v", err)
// 		}
//
// 		log.Println("Recreating offer")
// 		offer, err := peerConnection.CreateOffer(nil)
// 		if err != nil {
// 			t.Fatalf("%v", err)
// 		}
//
// 		log.Println("Set localDesc")
// 		err = peerConnection.SetLocalDescription(offer)
// 		if err != nil {
// 			panic(err)
// 		}
// 		log.Println("return offer")
// 		return cws.WSPacket{
// 			ID:   "init_webrtc",
// 			Data: gamertc.Encode(offer),
// 		}
// 	})
//
// 	<-handshakedone
// 	return client
// 	// If receive roomID, the server is running correctly
// }
//
// func TestSingleServerOneCoordinator(t *testing.T) {
// 	/*
// 		Case scenario:
// 		- A server X are initilized
// 		- Client join room with coordinator
// 		Expected behavior:
// 		- Room received not empty.
// 	*/
//
// 	oworker, obrowser := initCoordinator()
// 	defer obrowser.Close()
// 	defer oworker.Close()
//
// 	// Init worker
// 	worker := initWorker(t, oworker.URL)
// 	defer worker.Close()
//
// 	// connect overlord
// 	client := initClient(t, obrowser.URL)
// 	defer client.Close()
//
// 	fmt.Println("Sending start...")
// 	roomID := make(chan string)
// 	client.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      "",
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		roomID <- resp.RoomID
// 	})
//
// 	respRoomID := <-roomID
// 	if respRoomID == "" {
// 		fmt.Println("RoomID should not be empty")
// 		t.Fail()
// 	}
// 	time.Sleep(time.Second)
// 	fmt.Println("Done")
// }
//
// func TestTwoServerOneCoordinator(t *testing.T) {
// 	/*
// 	   Case scenario:
// 	   - Two server X, Y are initilized
// 	   - Client A creates a room on server X
// 	   - Client B creates a room on server Y
// 	   - Client B join a room created by A
// 	   Expected behavior:
// 	   - Bridge connection will be conducted between server Y and X
// 	   - Client B can join a room hosted on A
// 	*/
//
// 	oworker, obrowser := initCoordinator()
// 	defer obrowser.Close()
// 	defer oworker.Close()
//
// 	worker1 := initWorker(t, oworker.URL)
// 	defer worker1.Close()
//
// 	worker2 := initWorker(t, oworker.URL)
// 	defer worker2.Close()
//
// 	client1 := initClient(t, obrowser.URL)
// 	defer client1.Close()
//
// 	roomID := make(chan string)
// 	client1.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      "",
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		roomID <- resp.RoomID
// 	})
//
// 	remoteRoomID := <-roomID
// 	if remoteRoomID == "" {
// 		fmt.Println("RoomID should not be empty")
// 		t.Fail()
// 	}
// 	fmt.Println("Done create a room in server 1")
//
// 	// ------------------------------------
// 	// Client2 trying to create a random room and later join the the room on server1
// 	client2 := initClient(t, obrowser.URL)
// 	defer client2.Close()
// 	// Wait
// 	// Doing the same create local room.
// 	localRoomID := make(chan string)
// 	client2.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      "",
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		localRoomID <- resp.RoomID
// 	})
//
// 	<-localRoomID
//
// 	fmt.Println("Request the room from server 1", remoteRoomID)
// 	log.Println("Server2 trying to join server1 room")
// 	// After trying loging in to one session, login to other with the roomID
// 	bridgeRoom := make(chan string)
// 	client2.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      remoteRoomID,
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		bridgeRoom <- resp.RoomID
// 	})
//
// 	<-bridgeRoom
// 	//respRoomID := <-bridgeRoom
// 	//if respRoomID == "" {
// 	//fmt.Println("The room ID should be equal to the saved room")
// 	//t.Fail()
// 	//}
// 	// If receive roomID, the server is running correctly
// 	time.Sleep(time.Second)
// 	fmt.Println("Done")
// }
//
// func TestReconnectRoom(t *testing.T) {
// 	/*
// 	   Case scenario:
// 	   - A server X is initialized connecting to overlord
// 	   - Client A creates a room K on server X
// 	   - Server X is turned down, Client is closed
// 	   - Spawn a new server and a new client connecting to the same room K
// 	   Expected behavior:
// 	   - The game should be continue
// 	   TODO: Current test just make sure the game is running, not check if the game is the same
// 	*/
//
// 	oworker, obrowser := initCoordinator()
// 	defer obrowser.Close()
// 	defer oworker.Close()
//
// 	// Init worker
// 	worker := initWorker(t, oworker.URL)
//
// 	client := initClient(t, obrowser.URL)
//
// 	fmt.Println("Sending start...")
// 	roomID := make(chan string)
// 	client.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      "",
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		roomID <- resp.RoomID
// 	})
//
// 	saveRoomID := <-roomID
// 	if saveRoomID == "" {
// 		fmt.Println("RoomID should not be empty")
// 		t.Fail()
// 	}
//
// 	log.Println("Closing room and server")
// 	client.Close()
// 	worker.GetCoordinatorClient().Close()
// 	worker.Close()
//
// 	// Close server and reconnect
//
// 	log.Println("Server respawn")
// 	// Init slave server again
// 	worker = initWorker(t, oworker.URL)
// 	defer worker.Close()
//
// 	client = initClient(t, obrowser.URL)
// 	defer client.Close()
//
// 	fmt.Println("Re-access room ", saveRoomID)
// 	roomID = make(chan string)
// 	client.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      saveRoomID,
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		roomID <- resp.RoomID
// 	})
//
// 	respRoomID := <-roomID
// 	if respRoomID == "" || respRoomID != saveRoomID {
// 		fmt.Println("The room ID should be equal to the saved room")
// 		t.Fail()
// 	}
//
// 	time.Sleep(time.Second)
// 	fmt.Println("Done")
// }
//
// func TestReconnectRoomNoLocal(t *testing.T) {
// 	/*
// 	   Case scenario:
// 	   - A server X is initialized connecting to overlord
// 	   - Client A creates a room K on server X
// 	   - Server X is turned down, Client is closed
// 	   - room on local is deleted
// 	   - Spawn a new server and a new client connecting to the same room K
// 	   Expected behavior:
// 	   - room on local storage is refetched from cloud storage
// 	   - The game should be continue where it is closed
// 	   TODO: Current test just make sure the game is running, not check if the game is the same
// 	*/
// 	// This test only run if GCP storage is set
//
// 	oworker, obrowser := initCoordinator()
// 	defer obrowser.Close()
// 	defer oworker.Close()
//
// 	// Init worker
// 	ggCredential := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
// 	if ggCredential == "" {
// 		return
// 	}
//
// 	worker := initWorker(t, oworker.URL)
//
// 	client := initClient(t, obrowser.URL)
//
// 	fmt.Println("Sending start...")
// 	roomID := make(chan string)
// 	client.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      "",
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		roomID <- resp.RoomID
// 	})
//
// 	saveRoomID := <-roomID
// 	if saveRoomID == "" {
// 		fmt.Println("RoomID should not be empty")
// 		t.Fail()
// 	}
//
// 	log.Println("Closing room and server")
// 	client.Close()
// 	worker.GetCoordinatorClient().Close()
// 	worker.Close()
// 	// Remove room on local
// 	path := util.GetSavePath(saveRoomID)
// 	os.Remove(path)
// 	// Expect Google cloud call
//
// 	// Close server and reconnect
//
// 	log.Println("Server respawn")
// 	// Init slave server again
// 	worker = initWorker(t, oworker.URL)
// 	defer worker.Close()
//
// 	client = initClient(t, obrowser.URL)
// 	defer client.Close()
//
// 	fmt.Println("Re-access room ", saveRoomID)
// 	roomID = make(chan string)
// 	client.Send(cws.WSPacket{
// 		ID:          "start",
// 		Data:        "Contra.nes",
// 		RoomID:      saveRoomID,
// 		PlayerIndex: 1,
// 	}, func(resp cws.WSPacket) {
// 		fmt.Println("RoomID:", resp.RoomID)
// 		roomID <- resp.RoomID
// 	})
//
// 	respRoomID := <-roomID
// 	if respRoomID == "" || respRoomID != saveRoomID {
// 		fmt.Println("The room ID should be equal to the saved room")
// 		t.Fail()
// 	}
//
// 	time.Sleep(time.Second)
// 	fmt.Println("Done")
// }

//func TestRejoinNoCoordinatorMultiple(t *testing.T) {
//[>
//Case scenario:
//- A server X without connecting to overlord
//- Client A keeps creating a new room
//Expected behavior:
//- The game should running normally
//*/

//// Init slave server
//s := initServer(t, nil)
//defer s.Close()

//fmt.Println("Num goRoutine before start: ", runtime.NumGoroutine())
//client := initClient(t, s.URL)
//for i := 0; i < 100; i++ {
//fmt.Println("Sending start...")
//// Keep starting game
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

//respRoomID := <-roomID
//if respRoomID == "" {
//fmt.Println("The room ID should be equal to the saved room")
//t.Fail()
//}
//}
//time.Sleep(time.Second)
//fmt.Println("Num goRoutine should be small: ", runtime.NumGoroutine())
//fmt.Println("Done")

//}

//func TestRejoinWithCoordinatorMultiple(t *testing.T) {
//[>
//Case scenario:
//- A server X is initialized connecting to overlord
//- Client A keeps creating a new room
//Expected behavior:
//- The game should running normally
//*/

//// Init slave server
//o := initCoordinator()
//defer o.Close()

//oconn := connectTestCoordinatorServer(t, o.URL)
//// Init slave server
//s := initServer(t, oconn)
//defer s.Close()

//fmt.Println("Num goRoutine before start: ", runtime.NumGoroutine())
//client := initClient(t, s.URL)
//for i := 0; i < 100; i++ {
//fmt.Println("Sending start...")
//// Keep starting game
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

//respRoomID := <-roomID
//if respRoomID == "" {
//fmt.Println("The room ID should be equal to the saved room")
//t.Fail()
//}
//}
//fmt.Println("Num goRoutine should be small: ", runtime.NumGoroutine())
//fmt.Println("Done")

//}
