package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/ui"
	"github.com/giongto35/cloud-game/util"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"
)

const (
	width        = 256
	height       = 240
	scale        = 3
	title        = "NES"
	gameboyIndex = "./static/gameboy.html"
	debugIndex   = "./static/index_ws.html"
)

var indexFN = gameboyIndex

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

var IsOverlord = false
var upgrader = websocket.Upgrader{}

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	imageChannel chan *image.RGBA
	inputChannel chan int
	// Done channel is to fire exit event when there is no webRTC session running
	Done chan struct{}

	rtcSessions  []*webrtc.WebRTC
	sessionsLock *sync.Mutex

	director *ui.Director
}

var rooms = map[string]*Room{}

// ID to peerconnection
var peerconnections = map[string]*webrtc.WebRTC{}
var port string = "8000"
var serverID = ""
var oclient *Client

func main() {
	fmt.Println("Usage: ./game [debug]")
	if len(os.Args) > 1 {
		// debug
		indexFN = debugIndex
		fmt.Println("Use debug version")
	}
	if len(os.Args) >= 3 {
		if os.Args[2] == "overlord" {
			IsOverlord = true
		}
		fmt.Println("Running as overlord ")
	}
	if len(os.Args) >= 4 {
		port = os.Args[3]
	}

	rand.Seed(time.Now().UTC().UnixNano())
	rooms = map[string]*Room{}

	// ignore origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/", getWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/ws", ws)

	if !IsOverlord {
		oclient = NewOverlordClient()
	}

	log.Println("oclient ", oclient)
	if !IsOverlord {
		fmt.Println("http://localhost:8000")
		http.ListenAndServe(":"+port, nil)
	} else {
		fmt.Println("http://localhost:9000")
		// Overlord expose one more path for handle overlord connections
		http.HandleFunc("/wso", wso)
		http.ListenAndServe(":9000", nil)
	}
}

func getWeb(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadFile(indexFN)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
}

// init initilizes a room returns roomID
func initRoom(roomID, gameName string) string {
	// if no roomID is given, generate it
	if roomID == "" {
		roomID = generateRoomID()
	}
	log.Println("Init new room", roomID)
	imageChannel := make(chan *image.RGBA, 100)
	inputChannel := make(chan int, 100)

	// create director
	director := ui.NewDirector(roomID, imageChannel, inputChannel)

	room := &Room{
		imageChannel: imageChannel,
		inputChannel: inputChannel,
		rtcSessions:  []*webrtc.WebRTC{},
		sessionsLock: &sync.Mutex{},
		director:     director,
		Done:         make(chan struct{}),
	}
	rooms[roomID] = room

	go room.start()
	go director.Start([]string{"games/" + gameName})

	return roomID
}

// isRoomRunning check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed, we can check if the sessions list is empty or not.
func isRoomRunning(roomID string) bool {
	// If no roomID is registered
	if _, ok := rooms[roomID]; !ok {
		return false
	}

	// If there is running session
	for _, s := range rooms[roomID].rtcSessions {
		if !s.IsClosed() {
			return true
		}
	}
	return false
}

// startSession handles one session call
func startSession(webRTC *webrtc.WebRTC, gameName string, roomID string, playerIndex int) (rRoomID string, isNewRoom bool) {
	isNewRoom = false
	cleanSession(webRTC)
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !isRoomRunning(roomID) {
		roomID = initRoom(roomID, gameName)
		isNewRoom = true
	}

	// TODO: Might have race condition
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	room := rooms[roomID]

	webRTC.AttachRoomID(roomID)
	go startWebRTCSession(room, webRTC, playerIndex)

	return roomID, isNewRoom
}

// Session represents a session connected from the browser to the current server
type Session struct {
	client         *Client
	peerconnection *webrtc.WebRTC
	ServerID       string
}

// Handle normal traffic (from browser to host)
func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}
	defer c.Close()
	var gameName string
	var roomID string
	var playerIndex int

	// Create connection to overlord
	client := NewClient(c)
	sessionID := strconv.Itoa(rand.Int())

	wssession := &Session{
		client:         client,
		peerconnection: webrtc.NewWebRTC(),
		// The server session is maintaining
	}

	client.receive("initwebrtc", func(resp WSPacket) WSPacket {
		log.Println("Received user SDP")
		localSession, err := wssession.peerconnection.StartClient(resp.Data, width, height)
		if err != nil {
			log.Fatalln(err)
		}

		return WSPacket{
			ID:        "sdp",
			Data:      localSession,
			SessionID: sessionID,
		}
	})

	client.receive("save", func(resp WSPacket) (req WSPacket) {
		log.Println("Saving game state")
		req.ID = "save"
		req.Data = "ok"
		if roomID != "" {
			err = rooms[roomID].director.SaveGame()
			if err != nil {
				log.Println("[!] Cannot save game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}

		return req
	})

	client.receive("load", func(resp WSPacket) (req WSPacket) {
		log.Println("Loading game state")
		req.ID = "load"
		req.Data = "ok"
		if roomID != "" {
			err = rooms[roomID].director.LoadGame()
			if err != nil {
				log.Println("[!] Cannot load game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}

		return req
	})

	client.receive("start", func(resp WSPacket) (req WSPacket) {
		gameName = resp.Data
		roomID = resp.RoomID
		playerIndex = resp.PlayerIndex
		isNewRoom := false

		log.Println("Starting game")
		// If we are connecting to overlord, request serverID from roomID
		if oclient != nil {
			roomServerID := getServerIDOfRoom(oclient, roomID)
			log.Println("Server of RoomID ", roomID, " is ", roomServerID)
			if roomServerID != "" && wssession.ServerID != roomServerID {
				// TODO: Re -register
				go bridgeConnection(wssession, roomServerID, gameName, roomID, playerIndex)
				return
			}
		}

		roomID, isNewRoom = startSession(wssession.peerconnection, gameName, roomID, playerIndex)
		// Register room to overlord if we are connecting to overlord
		if isNewRoom && oclient != nil {
			oclient.send(WSPacket{
				ID:   "registerRoom",
				Data: roomID,
			}, nil)
		}
		req.ID = "start"
		req.RoomID = roomID
		req.SessionID = sessionID

		return req
	})

	client.receive("candidate", func(resp WSPacket) (req WSPacket) {
		// Unuse code
		hi := pionRTC.ICECandidateInit{}
		err = json.Unmarshal([]byte(resp.Data), &hi)
		if err != nil {
			log.Println("[!] Cannot parse candidate: ", err)
		} else {
			// webRTC.AddCandidate(hi)
		}
		req.ID = "candidate"

		return req
	})

	client.listen()
}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	return roomID
}

func (r *Room) start() {
	// fanout Screen
	for {
		select {
		case <-r.Done:
			r.remove()
			return
		case image := <-r.imageChannel:
			//isRoomRunning := false

			yuv := util.RgbaToYuv(image)
			r.sessionsLock.Lock()
			for _, webRTC := range r.rtcSessions {
				// Client stopped
				if webRTC.IsClosed() {
					continue
				}

				// encode frame
				// fanout imageChannel
				if webRTC.IsConnected() {
					// NOTE: can block here
					webRTC.ImageChannel <- yuv
				}
				//isRoomRunning = true
			}
			r.sessionsLock.Unlock()
		}
	}
}

func (r *Room) remove() {
	log.Println("Closing room", r)
	r.director.Done <- struct{}{}
}

// startWebRTCSession fan-in of the same room to inputChannel
func startWebRTCSession(room *Room, webRTC *webrtc.WebRTC, playerIndex int) {
	inputChannel := room.inputChannel
	fmt.Println("room, inputChannel", room, inputChannel)
	for {
		select {
		case <-webRTC.Done:
			fmt.Println("One session closed")
			removeSession(webRTC, room)
		default:
		}
		// Client stopped
		if webRTC.IsClosed() {
			return
		}

		// encode frame
		if webRTC.IsConnected() {
			input := <-webRTC.InputChannel
			// the first 8 bits belong to player 1
			// the next 8 belongs to player 2 ...
			// We standardize and put it to inputChannel (16 bits)
			input = input << ((uint(playerIndex) - 1) * ui.NumKeys)
			inputChannel <- input
		}
	}
}

func cleanSession(w *webrtc.WebRTC) {
	room, ok := rooms[w.RoomID]
	if !ok {
		return
	}
	removeSession(w, room)
}

func removeSession(w *webrtc.WebRTC, room *Room) {
	room.sessionsLock.Lock()
	defer room.sessionsLock.Unlock()
	for i, s := range room.rtcSessions {
		if s == w {
			room.rtcSessions = append(room.rtcSessions[:i], room.rtcSessions[i+1:]...)
			break
		}
	}
	// If room has no sessions, close room
	if len(room.rtcSessions) == 0 {
		room.Done <- struct{}{}
	}
}

func getServerIDOfRoom(oc *Client, roomID string) string {
	packet := oc.syncSend(
		WSPacket{
			ID:   "getRoom",
			Data: roomID,
		},
	)

	return packet.Data
}

func bridgeConnection(session *Session, serverID string, gameName string, roomID string, playerIndex int) {
	log.Println("Bridging connection to other Host ", serverID)
	client := session.client
	// Ask client to init

	log.Println("Requesting offer to browser", serverID)
	resp := client.syncSend(WSPacket{
		ID:   "requestOffer",
		Data: "",
	})

	log.Println("Sending offer to overlord to relay message to target host", resp.TargetHostID)
	// Ask overlord to relay SDP packet to serverID
	resp.TargetHostID = serverID
	remoteTargetSDP := oclient.syncSend(resp)
	log.Println("Got back remote host SDP, sending to browser")
	// Send back remote SDP of remote server to browser
	//client.syncSend(WSPacket{
	//ID:   "sdp",
	//Data: remoteTargetSDP.Data,
	//})
	client.send(WSPacket{
		ID:   "sdp",
		Data: remoteTargetSDP.Data,
	}, nil)
	log.Println("Init session done, start game on target host")

	oclient.syncSend(WSPacket{
		ID:           "start",
		Data:         gameName,
		TargetHostID: serverID,
		RoomID:       roomID,
		PlayerIndex:  playerIndex,
	})
	log.Println("Game is started on remote host")
}

const overlordHost = "ws://localhost:9000/wso"

func createOverlordConnection() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(overlordHost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewOverlordClient() *Client {
	oc, err := createOverlordConnection()
	if err != nil {
		log.Println("Cannot connect to overlord")
		log.Println("Run as a single server")
		return nil
	}
	oclient := NewClient(oc)
	oclient.send(
		WSPacket{
			ID: "ping",
		},
		func(resp WSPacket) {
			log.Println("Received pong full flow")
		},
	)

	// Received from overlord the serverID
	oclient.receive(
		"serverID",
		func(response WSPacket) (request WSPacket) {
			// Stick session with serverID got from overlord
			log.Println("Received serverID ", response.Data)
			serverID = response.Data

			return EmptyPacket
		},
	)

	// Received from overlord the sdp. This is happens when bridging
	// TODO: refactor
	oclient.receive(
		"initwebrtc",
		func(resp WSPacket) (req WSPacket) {
			log.Println("Received a sdp request from overlord")
			log.Println("Start peerconnection from the sdp")
			peerconnection := webrtc.NewWebRTC()
			// init new peerconnection from sessionID
			localSession, err := peerconnection.StartClient(resp.Data, width, height)
			peerconnections[resp.SessionID] = peerconnection

			if err != nil {
				log.Fatalln(err)
			}

			return WSPacket{
				ID:   "sdp",
				Data: localSession,
			}
		},
	)

	// Received start from overlord. This is happens when bridging
	// TODO: refactor
	oclient.receive(
		"start",
		func(resp WSPacket) (req WSPacket) {
			log.Println("Received a start request from overlord")
			log.Println("Add the connection to current room on the host")

			peerconnection := peerconnections[resp.SessionID]
			roomID, isNewRoom := startSession(peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
			// Bridge always access to old room
			// TODO: log warn
			if isNewRoom == true {
				log.Fatal("Bridge should not spawn new room")
			}

			req.ID = "start"
			req.RoomID = roomID
			return req
		},
	)
	go oclient.listen()

	return oclient
}
