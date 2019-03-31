package main

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/giongto35/game-online/ui"
	"github.com/gorilla/mux"
	"github.com/poi5305/go-yuv2webRTC/screenshot"
	"github.com/poi5305/go-yuv2webRTC/webrtc"
)

var webRTC *webrtc.WebRTC
var width = 256
var height = 240

func init() {
}

func startGame(path string, imageChannel chan *image.RGBA) {
	ui.Run([]string{path}, imageChannel)
}

func main() {
	fmt.Println("http://localhost:8000")
	webRTC = webrtc.NewWebRTC()

	router := mux.NewRouter()
	router.HandleFunc("/", getWeb).Methods("GET")
	router.HandleFunc("/session", postSession).Methods("POST")

	go http.ListenAndServe(":8000", router)

	// start screenshot loop, wait for connection
	imageChannel := make(chan *image.RGBA, 2)
	go screenshotLoop(imageChannel)
	startGame("games/supermariobros.rom", imageChannel)
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

func randomImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{uint8(rand.Int31n(0xff)), uint8(rand.Int31n(0xff)), uint8(rand.Int31n(0xff)), 0xff - 1})
		}
	}

	return img
}

func screenshotLoop(imageChannel chan *image.RGBA) {
	for image := range imageChannel {
		if webRTC.IsConnected() {
			//rgbaImg := randomImage(width, height)
			yuv := screenshot.RgbaToYuv(image)
			webRTC.ImageChannel <- yuv
		}
		time.Sleep(10 * time.Millisecond)
	}
}
