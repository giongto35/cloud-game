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

func main() {
	fmt.Println("Usage: ./game [debug]")
	if len(os.Args) > 1 {
		// debug
		indexFN = debugIndex
		fmt.Println("Use debug version")
	}
	if len(os.Args) == 3 {
		if os.Args[3] == "overlord" {
			IsOverlord = true
		}
	}

	rand.Seed(time.Now().UTC().UnixNano())
	fmt.Println("http://localhost:8000")
	rooms = map[string]*Room{}

	// ignore origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	http.HandleFunc("/ws", ws)

	http.HandleFunc("/", getWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.ListenAndServe(":8000", nil)
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
func startSession(webRTC *webrtc.WebRTC, gameName string, roomID string, playerIndex int) string {
	cleanSession(webRTC)
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !isRoomRunning(roomID) {
		roomID = initRoom(roomID, gameName)
	}

	// TODO: Might have race condition
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	room := rooms[roomID]

	webRTC.AttachRoomID(roomID)
	go startWebRTCSession(room, webRTC, playerIndex)

	return roomID
}

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

	client := NewClient(c, webrtc.NewWebRTC())

	client.syncReceive("initwebrtc", func(req WSPacket) WSPacket {
		log.Println("Received user SDP")
		localSession, err := client.peerconnection.StartClient(req.Data, width, height)
		if err != nil {
			log.Fatalln(err)
		}

		return WSPacket{
			ID:   "sdp",
			Data: localSession,
		}
	})

	client.syncReceive("save", func(req WSPacket) (res WSPacket) {
		log.Println("Saving game state")
		res.ID = "save"
		res.Data = "ok"
		if roomID != "" {
			err = rooms[roomID].director.SaveGame()
			if err != nil {
				log.Println("[!] Cannot save game state: ", err)
				res.Data = "error"
			}
		} else {
			res.Data = "error"
		}

		return res
	})

	client.syncReceive("load", func(req WSPacket) (res WSPacket) {
		log.Println("Loading game state")
		res.ID = "load"
		res.Data = "ok"
		if roomID != "" {
			err = rooms[roomID].director.LoadGame()
			if err != nil {
				log.Println("[!] Cannot load game state: ", err)
				res.Data = "error"
			}
		} else {
			res.Data = "error"
		}

		return res
	})

	client.syncReceive("start", func(req WSPacket) (res WSPacket) {
		gameName = req.Data
		roomID = req.RoomID
		playerIndex = req.PlayerIndex
		//log.Println("Ping from server with game:", gameName)
		//res.ID = "pong"
		log.Println("Starting game")
		roomID = startSession(client.peerconnection, gameName, roomID, playerIndex)
		res.ID = "start"
		res.RoomID = roomID

		return res
	})

	client.syncReceive("candidate", func(req WSPacket) (res WSPacket) {
		// Unuse code
		hi := pionRTC.ICECandidateInit{}
		err = json.Unmarshal([]byte(req.Data), &hi)
		if err != nil {
			log.Println("[!] Cannot parse candidate: ", err)
		} else {
			// webRTC.AddCandidate(hi)
		}
		res.ID = "candidate"

		return res
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

//func (o *Overlord) isRemoteRoom(roomID string) bool {
//err := c.WriteMessage(websocket.TextMessage, []byte(stRes))
//o.ws.WriteMessage()
//_, message, err := o.ws.ReadMessage()
//if message == "isRemoteRoom" {
//return true
//}
//return false
//}
