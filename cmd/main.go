package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	log.Println("Listening at port: localhost:8000")
	err := http.ListenAndServe(":8000", nil)
	// Print err if overlord cannot launch
	if err != nil {
		log.Fatal(err)
	}
}

// initializeWorker setup a worker
func initializeWorker() {
	worker := worker.NewHandler(*config.OverlordHost, gamePath)

	defer func() {
		log.Println("Close worker")
		worker.Close()
	}()

	go worker.Run()
	port := 9000
	// It's recommend to run one worker on one instance. This logic is to make sure more than 1 workers still work
	for {
		log.Println("Listening at port: localhost:", port)
		//err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			port++
			continue
		}
		if port == 9100 {
			// Cannot find port
			return
		}

		l.Close()
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+strconv.Itoa(port), nil)
	}
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

	if *config.IsMonitor {
		go monitor()
	}
	// There are two server mode
	// Overlord is coordinator. If the OvelordHost Param is `overlord`, we spawn a new host as Overlord.
	// else we spawn new server as normal server connecting to OverlordHost.
	if *config.OverlordHost == "overlord" {
		log.Println("Running as overlord ")
		log.Println("Waiting for workers...")
		initilizeOverlord()
	} else {
		if strings.HasPrefix(*config.OverlordHost, "ws") && !strings.HasSuffix(*config.OverlordHost, "wso") {
			log.Fatal("Overlord connection is invalid. Should have the form `ws://.../wso`")
		}
		log.Println("Running as worker ")
		initializeWorker()
	}
}
