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
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/ui"
	"github.com/giongto35/cloud-game/util"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	pionRTC "github.com/pion/webrtc"
)

const gameboyIndex = "./static/gameboy.html"
const wsIndex = "./static/index_ws.html"

var width = 256
var height = 240
var indexFN string = gameboyIndex
var service string = "ws"

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

type IndexPageData struct {
	Service string
}

var upgrader = websocket.Upgrader{}

type WSPacket struct {
	ID          string `json:"id"`
	Data        string `json:"data"`
	RoomID      string `json:"room_id"`
	PlayerIndex int    `json:"player_index"`
}

type Room struct {
	imageChannel chan *image.RGBA
	inputChannel chan int
	rtcSessions  []*webrtc.WebRTC
}

var rooms map[string]*Room

func main() {
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

func startGame(path string, roomID string, imageChannel chan *image.RGBA, inputChannel chan int) {
	ui.Run([]string{path}, roomID, imageChannel, inputChannel)
}

func getWeb(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadFile(indexFN)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
}

// init initilize room returns roomID
func initRoom(roomID, gameName string) string {
	roomID = generateRoomID()
	imageChannel := make(chan *image.RGBA, 100)
	inputChannel := make(chan int, 100)
	rooms[roomID] = &Room{
		imageChannel: imageChannel,
		inputChannel: inputChannel,
		rtcSessions:  []*webrtc.WebRTC{},
	}
	go fanoutScreen(imageChannel, roomID)
	go startGame("games/"+gameName, roomID, imageChannel, inputChannel)

	return roomID
}

// startSession handles one session call
func startSession(webRTC *webrtc.WebRTC, gameName string, roomID string, playerIndex int) string {
	// If the roomID is empty, we spawn a new room
	if roomID == "" {
		roomID = initRoom(roomID, gameName)
	}

	// TODO: Might have race condition
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	faninInput(rooms[roomID].inputChannel, webRTC, playerIndex)

	return roomID
}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	log.Println("New Connection")
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
			res.ID = "start"
			res.RoomID = startSession(webRTC, gameName, roomID, playerIndex)

			isDone = true
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
		yuv := util.RgbaToYuv(image)
		for _, webRTC := range rooms[roomID].rtcSessions {
			// Client stopped
			if webRTC.IsClosed() {
				continue
			}

			// encode frame
			// fanout imageChannel
			if webRTC.IsConnected() {
				webRTC.ImageChannel <- yuv
			}
		}
	}
}

// faninInput fan-in of the same room to inputChannel
func faninInput(inputChannel chan int, webRTC *webrtc.WebRTC, playerIndex int) {
	go func() {
		for {
			// Client stopped
			if webRTC.IsClosed() {
				return
			}

			// encode frame
			if webRTC.IsConnected() {
				input := <-webRTC.InputChannel
				// the first 10 bits belong to player 1
				// the next 10 belongs to player 2 ...
				// We standardize and put it to inputChannel (20 bytes)
				input = input << ((uint(playerIndex) - 1) * ui.NumKeys / 2)
				inputChannel <- input
			}
		}
	}()
}
