package overlord

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/giongto35/cloud-game/cws"
	"github.com/gorilla/websocket"
)

type Server struct {
	roomToServer map[string]string
	// servers are the map serverID to server Client
	servers map[string]*cws.Client
}

var upgrader = websocket.Upgrader{}

func NewServer() *Server {
	return &Server{
		servers:      map[string]*cws.Client{},
		roomToServer: map[string]string{},
	}
}

// If it's overlord, handle overlord connection (from host to overlord)
func (o *Server) WSO(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Connected")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Overlord: [!] WS upgrade:", err)
		return
	}
	defer c.Close()

	// Register new server
	serverID := strconv.Itoa(rand.Int())
	log.Println("Overlord: A new server connected to Overlord", serverID)

	// Register to servers map the client connection
	client := cws.NewClient(c)
	o.servers[serverID] = client

	//wssession := &Session{
	//client:         client,
	//peerconnection: webrtc.NewWebRTC(),
	//// The server session is maintaining
	//}

	// Sendback the ID to server
	client.Send(
		cws.WSPacket{
			ID:   "serverID",
			Data: serverID,
		},
		nil,
	)

	// registerRoom event from a server, when server created a new room.
	// RoomID is global so it is managed by overlord.
	client.Receive("registerRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received registerRoom ", resp.Data, serverID)
		o.roomToServer[resp.Data] = serverID
		return cws.WSPacket{
			ID: "registerRoom",
		}
	})

	// getRoom returns the server ID based on requested roomID.
	client.Receive("getRoom", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received a getroom request")
		return cws.WSPacket{
			ID:   "getRoom",
			Data: o.roomToServer[resp.Data],
		}
	})

	// Relay message from server to other target server
	// TODO: Generalize
	client.Receive("initwebrtc", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received a relay sdp request from a host")
		// TODO: Abstract
		if resp.TargetHostID != serverID {
			log.Println("Overlord: Sending relay sdp to target host")
			// relay SDP to target host and get back sdp
			// TODO: Async
			sdp := o.servers[resp.TargetHostID].SyncSend(
				resp,
			)

			return sdp
		}
		log.Println("Overlord: Target host is overlord itself: start peerconnection")
		// If the target is in master
		// start by its old
		//localSession, err := wssession.peerconnection.StartClient(resp.Data, width, height)
		//if err != nil {
		//log.Fatalln(err)
		//}

		//return cws.WSPacket{
		//ID:   "sdp",
		//Data: localSession,
		//}
		return cws.EmptyPacket
	})

	// TODO: use relay ID type
	// TODO: Merge sdp and start
	client.Receive("start", func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Overlord: Received a relay start request from a host")
		// TODO: Abstract
		if resp.TargetHostID != serverID {
			// relay start to target host
			log.Println("Sending to target host", resp.TargetHostID, " ", resp)
			// TODO: Async
			resp := o.servers[resp.TargetHostID].SyncSend(
				resp,
			)

			return resp
		}
		log.Println("Overlord: Target host is overlord itself: start game")
		//// If the target is in master
		//// start by its old
		//roomID, isNewRoom := startSession(wssession.peerconnection, resp.Data, resp.RoomID, resp.PlayerIndex)
		//// Bridge always access to old room
		//// TODO: log warn
		//if isNewRoom == true {
		//log.Fatal("Bridge should not spawn new room")
		//}

		//return cws.WSPacket{
		//ID:     "start",
		//RoomID: roomID,
		//}
		return cws.EmptyPacket
	})

	client.Listen()
}
