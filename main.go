package main

import (
	pionRTC "github.com/pions/webrtc"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"

	"time"

	"github.com/giongto35/cloud-game/ui"
	"github.com/giongto35/cloud-game/util"
	"github.com/giongto35/cloud-game/webrtc"

	// "github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"encoding/json"
)

// var webRTC *webrtc.WebRTC
var width = 256
var height = 240
// var gameName = "supermariobros.rom"
var gameName string

// var FPS = 60

var upgrader = websocket.Upgrader{}

type WSPacket struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

func init() {
}

func startGame(path string, imageChannel chan *image.RGBA, inputChannel chan int, webRTC *webrtc.WebRTC) {
	ui.Run([]string{path}, imageChannel, inputChannel, webRTC)
}

func main() {
	fmt.Println("http://localhost:8000")
	fmt.Println(time.Now().UnixNano())

	// router := mux.NewRouter()
	// router.HandleFunc("/", getWeb).Methods("GET")
	// router.HandleFunc("/session", postSession).Methods("POST")
	// http.ListenAndServe(":8000", router)

	// ignore origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/", getWeb)
	http.HandleFunc("/ws", ws)

	http.ListenAndServe(":8000", nil)
}

func getWeb(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadFile("./index.html")
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
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
			imageChannel := make(chan *image.RGBA, 100)
			go screenshotLoop(imageChannel, webRTC)
			go startGame("games/" + gameName, imageChannel, webRTC.InputChannel, webRTC)
			res.ID = "start"
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

func postSession(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	r.Body.Close()

	webRTC := webrtc.NewWebRTC()

	localSession, err := webRTC.StartClient(string(bs), width, height)
	if err != nil {
		log.Fatalln(err)
	}

	imageChannel := make(chan *image.RGBA, 100)
	go screenshotLoop(imageChannel, webRTC)
	go startGame("games/"+gameName, imageChannel, webRTC.InputChannel, webRTC)

	w.Write([]byte(localSession))
}

// func screenshotLoop(imageChannel chan *image.RGBA) {
func screenshotLoop(imageChannel chan *image.RGBA, webRTC *webrtc.WebRTC) {
	for image := range imageChannel {
		// encode frame
		if webRTC.IsConnected() {
			yuv := util.RgbaToYuv(image)
			webRTC.ImageChannel <- yuv
		} else {
			break
		}
	}
}
