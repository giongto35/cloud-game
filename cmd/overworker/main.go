package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/worker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// initializeWorker setup a worker
func initializeWorker() {
	worker := worker.NewHandler(*config.OverlordHost)

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
		if port == 9000 {
			// only turn on metric for the first worker to avoid overlap
			http.Handle("/metrics", promhttp.Handler())
		}

		// echo endpoint is where user will request to test latency
		http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			fmt.Fprintf(w, "echo")
		})

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

	log.Println("Running as worker ")
	initializeWorker()
}
