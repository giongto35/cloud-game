package main

import (
	"math/rand"
	"os"
	"strconv"
	"time"

	pionRTC "github.com/pion/webrtc"

	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/ui"
	"github.com/giongto35/cloud-game/util"
	"github.com/giongto35/cloud-game/webrtc"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"encoding/json"
)

// var webRTC *webrtc.WebRTC
var width = 256
var height = 240
var index string = "./index_http.html"

var upgrader = websocket.Upgrader{}

type WSPacket struct {
	ID     string `json:"id"`
	Data   string `json:"data"`
	RoomID string `json:"room_id"`
}

type Room struct {
	imageChannel chan *image.RGBA
	inputChannel chan int
	rtcSessions  []*webrtc.WebRTC
}

var rooms map[string]*Room

func init() {
}

func startGame(path string, imageChannel chan *image.RGBA, inputChannel chan int, webRTC *webrtc.WebRTC) {
	ui.Run([]string{path}, imageChannel, inputChannel, webRTC)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	fmt.Printf("Usage: %s [ws]\n", os.Args[0])
	fmt.Println("http://localhost:8000")
	rooms = map[string]*Room{}
	if len(os.Args) > 1 {
		log.Println("Using websocket")
		index = "./index_ws.html"

		// ignore origin
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		http.HandleFunc("/", getWeb)
		http.HandleFunc("/ws", ws)
		http.ListenAndServe(":8000", nil)
	} else {
		log.Println("Using http")
		router := mux.NewRouter()
		router.HandleFunc("/", getWeb).Methods("GET")
		router.HandleFunc("/session", postSession).Methods("POST")
		http.ListenAndServe(":8000", router)
	}
}

func getWeb(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadFile(index)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
}

func startSession(webRTC *webrtc.WebRTC, gameName string, roomID string) string {
	if roomID == "" {
		roomID = generateRoomID()
		imageChannel := make(chan *image.RGBA, 100)
		inputChannel := make(chan int, 100)
		rooms[roomID] = &Room{
			imageChannel: imageChannel,
			inputChannel: inputChannel,
			rtcSessions:  []*webrtc.WebRTC{},
		}
		go fanoutScreen(imageChannel, roomID)
		go startGame("games/"+gameName, imageChannel, inputChannel, webRTC)
	}

	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	faninInput(rooms[roomID].inputChannel, webRTC)

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

	for !isDone {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("[!] read:", err)
			break
		}

		req := WSPacket{}
		err = json.Unmarshal(message, &req)
		if err != nil {
			log.Println("[!] json unmarshal:", err)
			break
		}
		// log.Println(req)

		// connectivity
		res := WSPacket{}
		switch req.ID {
		case "ping":
			gameName = req.Data
			roomID = req.RoomID
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
			res.RoomID = startSession(webRTC, gameName, roomID)

			isDone = true
		}

		stRes, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal:", err)
		}

		err = c.WriteMessage(mt, []byte(stRes))
		if err != nil {
			log.Println("write:", err)
			break
		}

	}
}

type SessionPacket struct {
	Game   string `json:"game"`
	RoomID string `json:"room_id"`
	SDP    string `json:"sdp"`
}

func postSession(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	r.Body.Close()

	var postPacket SessionPacket
	err = json.Unmarshal(bs, &postPacket)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Got session with game request:", postPacket.Game)

	webRTC := webrtc.NewWebRTC()

	localSession, err := webRTC.StartClient(postPacket.SDP, width, height)
	if err != nil {
		log.Fatalln(err)
	}

	roomID := postPacket.RoomID
	if roomID == "" {
		fmt.Println("Init Room")
		//generate new room
		roomID = generateRoomID()

		imageChannel := make(chan *image.RGBA, 100)
		inputChannel := make(chan int, 100)
		rooms[roomID] = &Room{
			imageChannel: imageChannel,
			inputChannel: inputChannel,
			rtcSessions:  []*webrtc.WebRTC{},
		}
		go fanoutScreen(imageChannel, roomID)
		go startGame("games/"+postPacket.Game, imageChannel, inputChannel, webRTC)
		// fanin input channel
		// fanout output channel
	} else {
		// if there is room, reuse image channel, add webRTC session
	}
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	faninInput(rooms[roomID].inputChannel, webRTC)

	res := SessionPacket{
		SDP:    localSession,
		RoomID: roomID,
	}
	stRes, err := json.Marshal(res)
	if err != nil {
		log.Println("json marshal:", err)
	}

	//w.Write([]byte(localSession))
	w.Write(stRes)
}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	return roomID
}

// func fanoutScreen(imageChannel chan *image.RGBA) {
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

// func faninInput(input chan int) {
func faninInput(inputChannel chan int, webRTC *webrtc.WebRTC) {
	go func() {
		for {
			fmt.Println("spawning")
			// Client stopped
			if webRTC.IsClosed() {
				return
			}

			// encode frame
			if webRTC.IsConnected() {
				input := <-webRTC.InputChannel
				fmt.Println("received input", input)
				inputChannel <- input
			}
		}
	}()
}
