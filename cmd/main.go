package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/handler"
	"github.com/giongto35/cloud-game/overlord"
	"github.com/gorilla/websocket"
)

const (
	gameboyIndex = "./static/gameboy.html"
	debugIndex   = "./static/index_ws.html"
)

var indexFN = gameboyIndex

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

var IsOverlord = false
var upgrader = websocket.Upgrader{}

func main() {
	flag.Parse()
	log.Println("Usage: ./game [debug]")
	if *config.IsDebug {
		// debug
		indexFN = debugIndex
		log.Println("Use debug version")
	}

	if *config.OverlordHost == "overlord" {
		log.Println("Running as overlord ")
		IsOverlord = true
	} else {
		if strings.HasPrefix(*config.OverlordHost, "ws") && !strings.HasSuffix(*config.OverlordHost, "wso") {
			log.Fatal("Overlord connection is invalid. Should have the form `ws://.../wso`")
		}
		log.Println("Running as slave ")
		IsOverlord = false
	}

	handler, err := handler.NewHandler(IsOverlord)
	rand.Seed(time.Now().UTC().UnixNano())

	// ignore origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/", handler.GetWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/ws", handler.WS)

	if !IsOverlord {
		conn, err := createOverlordConnection()
		if err != nil {
			log.Println("Cannot connect to overlord")
			log.Println("Run as a single server")
			oclient = nil
		} else {
			oclient = NewOverlordClient(conn)
		}
	}

	if !IsOverlord {
		log.Println("http://localhost:" + *config.Port)
		http.ListenAndServe(":"+*config.Port, nil)
	} else {
		log.Println("http://localhost:9000")
		// Overlord expose one more path for handle overlord connections
		http.HandleFunc("/wso", overlord.WSO)
		http.ListenAndServe(":9000", nil)
	}
}
