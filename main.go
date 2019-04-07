package main

import (
	"math/rand"
	"os"
	"strconv"

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
	ID   string `json:"id"`
	Data string `json:"data"`
}

type Room struct {
	imageChannel chan *image.RGBA
	rtcSessions  []*webrtc.WebRTC
}

var rooms map[string]Room

func init() {
}

func startGame(path string, imageChannel chan *image.RGBA, inputChannel chan int, webRTC *webrtc.WebRTC) {
	ui.Run([]string{path}, imageChannel, inputChannel, webRTC)
}

func main() {
	fmt.Printf("Usage: %s [ws]\n", os.Args[0])
	fmt.Println("http://localhost:8000")
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
			//res.RoomID = generateRoomID()

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
			go startGame("games/"+gameName, imageChannel, webRTC.InputChannel, webRTC)
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

	if postPacket.RoomID == "" {
		imageChannel := make(chan *image.RGBA, 100)
		rooms[postPacket.RoomID] = Room{
			imageChannel: imageChannel,
			rtcSessions:  []*webrtc.WebRTC{},
		}
		go screenshotLoop(imageChannel, room)
	} else {
		// if there is room, reuse image channel
		rooms[postPacket.RoomID] = append(rooms[postPacket.RoomID], webRTC)
	}
	go startGame("games/"+postPacket.Game, imageChannel, webRTC.InputChannel, webRTC)

	w.Write([]byte(localSession))
}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	for len(roomID) < 16 {
		roomID = "0" + roomID
	}
	fmt.Println(roomID)
	return roomID
}

// func screenshotLoop(imageChannel chan *image.RGBA) {
func screenshotLoop(imageChannel chan *image.RGBA, roomID string) {
	for image := range imageChannel {
		yuv := util.RgbaToYuv(image)
		for _, webRTC := range rooms[roomID].rtcSessions {
			// Client stopped
			if webRTC.IsClosed() {
				continue
			}

			// encode frame
			if webRTC.IsConnected() {
				webRTC.ImageChannel <- yuv
			}
		}
		// time.Sleep(10 * time.Millisecond)
		// time.Sleep(time.Duration(1000 / FPS) * time.Millisecond)
	}
}
