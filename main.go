package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/game-online/ui"
	"github.com/giongto35/game-online/util"
	"github.com/giongto35/game-online/webrtc"
	"github.com/gorilla/mux"
)

var webRTC *webrtc.WebRTC
var width = 256
var height = 240
var gameName = "supermariobros.rom"

func init() {
}

func startGame(path string, imageChannel chan *image.RGBA, inputChannel chan int) {
	ui.Run([]string{path}, imageChannel, inputChannel)
}

func main() {
	imageChannel := make(chan *image.RGBA, 2)
	fmt.Println("http://localhost:8000")
	webRTC = webrtc.NewWebRTC()

	router := mux.NewRouter()
	router.HandleFunc("/", getWeb).Methods("GET")
	router.HandleFunc("/session", postSession).Methods("POST")

	go http.ListenAndServe(":8000", router)

	// start screenshot loop, wait for connection
	go screenshotLoop(imageChannel)
	startGame("games/"+gameName, imageChannel, webRTC.InputChannel)
	time.Sleep(time.Minute)
}

func getWeb(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadFile("./index.html")
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
}

func postSession(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	r.Body.Close()

	localSession, err := webRTC.StartClient(string(bs), width, height)
	if err != nil {
		log.Fatalln(err)
	}

	w.Write([]byte(localSession))
}

func screenshotLoop(imageChannel chan *image.RGBA) {
	for image := range imageChannel {
		if webRTC.IsConnected() {
			yuv := util.RgbaToYuv(image)
			webRTC.ImageChannel <- yuv
		}
		time.Sleep(10 * time.Millisecond)
	}
}
