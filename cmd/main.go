package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/overlord"
	"github.com/giongto35/cloud-game/worker"
	"github.com/gorilla/websocket"
)

const gamePath = "games"

// Time allowed to write a message to the peer.
var upgrader = websocket.Upgrader{}

// initilizeOverlord setup an overlord server
func initilizeOverlord() {
	overlord := overlord.NewServer()

	http.HandleFunc("/", overlord.GetWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// browser facing port
	go func() {
		http.HandleFunc("/ws", overlord.WS)
	}()

	// worker facing port
	http.HandleFunc("/wso", overlord.WSO)
	http.ListenAndServe(":8000", nil)
}

// initializeWorker setup a worker
func initializeWorker() {
	worker := worker.NewHandler(*config.OverlordHost, gamePath)

	defer func() {
		log.Println("Close worker")
		worker.Close()
	}()

	go worker.Run()
	port := rand.Int()%100 + 8000
	log.Println("Listening at port: localhost:", port)
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}

func monitor() {
	c := time.Tick(time.Second)
	for range c {
		log.Printf("#goroutines: %d\n", runtime.NumGoroutine())
	}
}

func main() {
	flag.Parse()

	rand.Seed(time.Now().UTC().UnixNano())

	//if *config.IsMonitor {
	go monitor()
	//}
	// There are two server mode
	// Overlord is coordinator. If the OvelordHost Param is `overlord`, we spawn a new host as Overlord.
	// else we spawn new server as normal server connecting to OverlordHost.
	if *config.OverlordHost == "overlord" {
		log.Println("Running as overlord ")
		initilizeOverlord()
	} else {
		if strings.HasPrefix(*config.OverlordHost, "ws") && !strings.HasSuffix(*config.OverlordHost, "wso") {
			log.Fatal("Overlord connection is invalid. Should have the form `ws://.../wso`")
		}
		log.Println("Running as worker ")
		initializeWorker()
	}
}
