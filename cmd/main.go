package main

import (
	"flag"
	"log"
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

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

var upgrader = websocket.Upgrader{}

// initilizeOverlord setup an overlord server
func initilizeOverlord() {
	overlord := overlord.NewServer()

	log.Println("http://localhost:9000")

	// Can consider Overlord works as server but it is complicated
	http.HandleFunc("/wso", overlord.WSO)
	http.ListenAndServe(":9000", nil)
}

func createOverlordConnection() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(*config.OverlordHost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// initializeServer setup a server
func initializeServer() {
	conn, err := createOverlordConnection()
	if err != nil {
		log.Println("Cannot connect to overlord")
		log.Println("Run as a single server")
	}

	handler := handler.NewHandler(conn, *config.IsDebug)

	// ignore origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/", handler.GetWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/ws", handler.WS)

	log.Println("http://localhost:" + *config.Port)
	http.ListenAndServe(":"+*config.Port, nil)
}

func main() {
	flag.Parse()
	log.Println("Usage: ./game [-debug]")

	if *config.OverlordHost == "overlord" {
		log.Println("Running as overlord ")
		initilizeOverlord()
	} else {
		if strings.HasPrefix(*config.OverlordHost, "ws") && !strings.HasSuffix(*config.OverlordHost, "wso") {
			log.Fatal("Overlord connection is invalid. Should have the form `ws://.../wso`")
		}
		log.Println("Running as slave ")
		initializeServer()
	}
}
