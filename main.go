package main

import (
	"os"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/ui"
	"github.com/giongto35/cloud-game/util"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"

	"github.com/hraban/opus"
)

const (
	width  = 256
	height = 240
	scale  = 3
	title  = "NES"
	gameboyIndex = "./static/gameboy.html"
	debugIndex = "./static/index_ws.html"
)

var indexFN = gameboyIndex

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

var upgrader = websocket.Upgrader{}

type WSPacket struct {
	ID          string `json:"id"`
	Data        string `json:"data"`
	RoomID      string `json:"room_id"`
	PlayerIndex int    `json:"player_index"`
}

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	imageChannel chan *image.RGBA
	audioChanel chan float32
	inputChannel chan int
	// closedChannel is to fire exit event when there is no webRTC session running
	closedChannel chan bool

	rtcSessions []*webrtc.WebRTC

	director *ui.Director
}

var rooms map[string]*Room

func main() {
	fmt.Println("Usage: ./game [debug]")
	if len(os.Args) > 1  {
		// debug
		indexFN = debugIndex
		fmt.Println("Use debug version")
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
	imageChannel := make(chan *image.RGBA, 100)
	audioChannel := make(chan float32, 48000)
	inputChannel := make(chan int, 100)
	closedChannel := make(chan bool)

	// create director
	director := ui.NewDirector(roomID, imageChannel, audioChannel, inputChannel, closedChannel)
	
	rooms[roomID] = &Room{
		imageChannel:  imageChannel,
		audioChanel:   audioChannel,
		inputChannel:  inputChannel,
		closedChannel: closedChannel,
		rtcSessions:   []*webrtc.WebRTC{},
		director:      director,
	}

	go fanoutScreen(imageChannel, roomID)
	go fanoutAudio(audioChannel, roomID)
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
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !isRoomRunning(roomID) {
		roomID = initRoom(roomID, gameName)
	}

	// TODO: Might have race condition
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	go faninInput(rooms[roomID].inputChannel, webRTC, playerIndex)

	return roomID
}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}
	defer c.Close()

	log.Println("New ws connection")
	webRTC := webrtc.NewWebRTC()

	// streaming game

	// start new games and webrtc stuff?
	isDone := false

	var gameName string
	var roomID string
	var playerIndex int

	for !isDone {
		c.SetReadDeadline(time.Now().Add(readWait))
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("[!] read:", err)
			break
		}

		req := WSPacket{}
		res := WSPacket{}

		err = json.Unmarshal(message, &req)
		if err != nil {
			log.Println("[!] json unmarshal:", err)
			break
		}

		// SDP connection initializations follows WebRTC convention
		// https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Protocols
		switch req.ID {
		case "ping":
			gameName = req.Data
			roomID = req.RoomID
			playerIndex = req.PlayerIndex
			log.Println("Ping from server with game:", gameName)
			res.ID = "pong"

		case "sdp":
			log.Println("Received user SDP")
			localSession, err := webRTC.StartClient(req.Data, width, height)
			if err != nil {
				log.Fatalln(err)
			}

			res.ID = "sdp"
			res.Data = localSession

		case "candidate":
			// Unuse code
			hi := pionRTC.ICECandidateInit{}
			err = json.Unmarshal([]byte(req.Data), &hi)
			if err != nil {
				log.Println("[!] Cannot parse candidate: ", err)
			} else {
				// webRTC.AddCandidate(hi)
			}
			res.ID = "candidate"

		case "start":
			log.Println("Starting game")
			roomID = startSession(webRTC, gameName, roomID, playerIndex)
			res.ID = "start"
			res.RoomID = roomID

			// maybe we wont close websocket
			// isDone = true
		
		case "save":
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

		case "load":
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
		}

		stRes, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal:", err)
		}

		c.SetWriteDeadline(time.Now().Add(writeWait))
		err = c.WriteMessage(mt, []byte(stRes))
		if err != nil {
			log.Println("write:", err)
			break
		}

	}
}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	return roomID
}

// fanoutScreen fanout outputs to all webrtc in the same room
func fanoutScreen(imageChannel chan *image.RGBA, roomID string) {
	for image := range imageChannel {
		isRoomRunning := false

		yuv := util.RgbaToYuv(image)
		for _, webRTC := range rooms[roomID].rtcSessions {
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
			isRoomRunning = true
		}

		if isRoomRunning == false {
			log.Println("Closed room from screen routine", roomID)
			rooms[roomID].closedChannel <- true
		}
	}
}

// fanoutAudio fanout outputs to all webrtc in the same room
func fanoutAudio(audioChanel chan float32, roomID string) {
	enc, err := opus.NewEncoder(48000, 1, opus.AppVoIP)
	if err != nil {
		log.Println("[!] Cannot create audio encoder")
		return
	}
	pcm := make([]float32, 120)
	c := 0

	for audio := range audioChanel {
		if c >= cap(pcm) {
			data := make([]byte, 1000)
			n, err := enc.EncodeFloat32(pcm, data)
			if err != nil {
				log.Println("[!] Failed to decode")
				continue
			}
			data = data[:n]

			isRoomRunning := false
			for _, webRTC := range rooms[roomID].rtcSessions {
				// Client stopped
				if webRTC.IsClosed() {
					continue
				}
	
				// encode frame
				// fanout imageChannel
				if webRTC.IsConnected() {
					// NOTE: can block here
					webRTC.AudioChannel <- data
				}
				isRoomRunning = true
			}

			if isRoomRunning == false {
				log.Println("Closed room from audio routine", roomID)
				rooms[roomID].closedChannel <- true
			}

			c = 0
		} else {
			pcm[c] = audio
			c++
		}


	}
}

// faninInput fan-in of the same room to inputChannel
func faninInput(inputChannel chan int, webRTC *webrtc.WebRTC, playerIndex int) {
	for {
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
