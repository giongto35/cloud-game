package main

import (
	"github.com/gorilla/websocket"
)

const overlordHost = "http://localhost:9000"

type Overlord struct {
	ws websocket.Conn
}

//func createOverlordClient() websocket.Conn {
//signal.Notify(interrupt, os.Interrupt)

////u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
////log.Printf("connecting to %s", u.String())

//c, _, err := websocket.DefaultDialer.Dial(overlordHost, nil)
//if err != nil {
//log.Fatal("dial:", err)
//}
//overlord := &Overlord{
//ws: c,
//}

//return overlord
//}
