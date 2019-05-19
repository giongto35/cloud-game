package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"runtime"
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

func createOverlordConnection() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(*config.OverlordHost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// initilizeOverlord setup an overlord server
func initilizeOverlord() {
	overlord := overlord.NewServer()

	log.Println("http://localhost:9000")

	http.HandleFunc("/", overlord.GetWeb)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// browser facing port
	go func() {
		http.HandleFunc("/ws", overlord.WS)
		http.ListenAndServe(":8000", nil)
	}()

	// worker facing port
	http.HandleFunc("/wso", overlord.WSO)
	http.ListenAndServe(":9000", nil)

	log.Println("http://localhost:" + *config.Port)
}

// initializeWorker setup a worker
func initializeWorker() {
	conn, err := createOverlordConnection()
	if err != nil {
		log.Println("Cannot connect to overlord")
		log.Println("Run as a single server")
	}

	worker := worker.NewHandler(conn, *config.IsDebug, gamePath)

	defer func() {
		log.Println("Close worker")
		worker.Close()
	}()
	worker.Run()
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
