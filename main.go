package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"

	// "time"

	"github.com/giongto35/game-online/ui"
	"github.com/giongto35/game-online/util"
	"github.com/giongto35/game-online/webrtc"

	// "github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"encoding/json"
)

// var webRTC *webrtc.WebRTC
var width = 256
var height = 240
var gameName = "supermariobros.rom"

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
	// imageChannel := make(chan *image.RGBA, 100)
	fmt.Println("http://localhost:8000")
	// webRTC = webrtc.NewWebRTC()

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

	webRTC := webrtc.NewWebRTC()
	localSession, err := webRTC.StartClient(width, height)
	if err != nil {
		log.Fatalln(err)
	}

	// streaming game
	// imageChannel := make(chan *image.RGBA, 100)
	// go screenshotLoop(imageChannel, webRTC)
	// go startGame("games/" + gameName, imageChannel, webRTC.InputChannel, webRTC)

	// start new games and webrtc stuff?
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		req := WSPacket{}
		err = json.Unmarshal(message, &req)
		if err != nil {
			log.Println("json unmarshal:", err)
			break
		}
		log.Println(req)

		// connectivity
		res := WSPacket{}
		switch req.ID {
		case "ping":
			res.ID = "pong"

		case "sdp":
			webRTC.SetRemoteSession(res.Data)
			res.ID = "sdp"
			res.Data = localSession

		case "candidate":
			res.ID = "candidate"
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

// func postSession(w http.ResponseWriter, r *http.Request) {
// 	bs, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	r.Body.Close()

// 	webRTC := webrtc.NewWebRTC()

// 	localSession, err := webRTC.StartClient(string(bs), width, height)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	imageChannel := make(chan *image.RGBA, 100)
// 	go screenshotLoop(imageChannel, webRTC)
// 	go startGame("games/"+gameName, imageChannel, webRTC.InputChannel, webRTC)

// 	w.Write([]byte(localSession))
// }

// func screenshotLoop(imageChannel chan *image.RGBA) {
func screenshotLoop(imageChannel chan *image.RGBA, webRTC *webrtc.WebRTC) {
	for image := range imageChannel {
		// Client stopped
		if webRTC.IsClosed() {
			break
		}

		// encode frame
		if webRTC.IsConnected() {
			yuv := util.RgbaToYuv(image)
			webRTC.ImageChannel <- yuv
		}
		// time.Sleep(10 * time.Millisecond)
		// time.Sleep(time.Duration(1000 / FPS) * time.Millisecond)
	}
}
