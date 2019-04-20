package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/giongto35/cloud-game/webrtc"
)

var roomToServer = map[string]string{}

// servers are the map serverID to server Client
var servers = map[string]*Client{}

// If it's overlord, handle overlord connection (from host to overlord)
func wso(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Connected")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}
	defer c.Close()

	// register new server
	serverID := strconv.Itoa(rand.Int())
	log.Println("A new server connected ", serverID)

	client := NewClient(c)
	servers[serverID] = client

	wssession := &Session{
		client:         client,
		peerconnection: webrtc.NewWebRTC(),
		// The server session is maintaining
	}

	client.send(
		WSPacket{
			ID:   "serverID",
			Data: serverID,
		},
		nil,
	)

	client.receive("ping", func(resp WSPacket) WSPacket {
		log.Println("received Ping, sending Pong")
		return WSPacket{
			ID: "pong",
		}
	})

	client.receive("registerRoom", func(resp WSPacket) WSPacket {
		log.Println("Received registerRoom ", resp.Data, serverID)
		roomToServer[resp.Data] = serverID
		return WSPacket{
			ID: "registerRoom",
		}
	})

	client.receive("getRoom", func(resp WSPacket) WSPacket {
		return WSPacket{
			ID:   "getRoom",
			Data: roomToServer[resp.Data],
		}
	})

	client.receive("initwebrtc", func(resp WSPacket) WSPacket {
		log.Println("Received a relay sdp request from a host")
		// TODO: Abstract
		if resp.TargetHostID != serverID {
			log.Println("sending relay sdp to target host", resp.TargetHostID)
			// relay SDP to target host and get back sdp
			// TODO: Async
			sdp := servers[resp.TargetHostID].syncSend(
				resp,
			)

			return sdp
		}
		log.Println("Target host is overlord itself: start peerconnection")
		// If the target is in master
		// start by its old
		localSession, err := wssession.peerconnection.StartClient(resp.Data, width, height)
		if err != nil {
			log.Fatalln(err)
		}

		return WSPacket{
			ID:   "sdp",
			Data: localSession,
		}
	})

	// TODO: use relay ID type
	// TODO: Merge sdp and start
	client.receive("start", func(resp WSPacket) WSPacket {
		log.Println("Received a relay start request from a host")
		// TODO: Abstract
		if resp.TargetHostID != serverID {
			// relay SDP to target host and get back sdp
			// TODO: Async
			resp := servers[resp.TargetHostID].syncSend(
				resp,
			)

			return resp
		}
		log.Println("Target host is overlord itself: start game")
		// If the target is in master
		// start by its old
		roomID, isNewRoom := startSession(wssession.peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
		// Bridge always access to old room
		// TODO: log warn
		if isNewRoom == true {
			log.Fatal("Bridge should not spawn new room")
		}

		return WSPacket{
			ID:     "start",
			RoomID: roomID,
		}
	})

	client.listen()
}
