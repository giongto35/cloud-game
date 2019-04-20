package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn

	peerconnection *webrtc.WebRTC

	// sendCallback is callback based on packetID
	sendCallback map[string]func(req WSPacket)
	// recvCallback is callback when receive based on ID of the packet
	recvCallback map[string]func(req WSPacket)
}

type WSPacket struct {
	ID   string `json:"id"`
	Data string `json:"data"`

	RoomID      string `json:"room_id"`
	PlayerIndex int    `json:"player_index"`

	TargetHostID string `json:"target_id"`
	PacketID     string
}

var EmptyPacket = WSPacket{}

func NewClient(conn *websocket.Conn, webrtc *webrtc.WebRTC) *Client {
	sendCallback := map[string]func(WSPacket){}
	recvCallback := map[string]func(WSPacket){}
	return &Client{
		conn: conn,

		peerconnection: webrtc,
		sendCallback:   sendCallback,
		recvCallback:   recvCallback,
	}
}

// send sends a packet and trigger callback when the packet comes back
func (c *Client) send(request WSPacket, callback func(response WSPacket)) {
	request.PacketID = strconv.Itoa(rand.Int())
	data, err := json.Marshal(request)
	if err != nil {
		return
	}

	c.conn.WriteMessage(websocket.TextMessage, data)
	if callback == nil {
		return
	}
	c.sendCallback[request.PacketID] = callback
}

// receive receive and response back
func (c *Client) receive(id string, f func(response WSPacket) (request WSPacket)) {
	c.recvCallback[id] = func(response WSPacket) {
		packet := f(response)
		// Add Meta data
		packet.PacketID = response.PacketID

		// Skip rqeuest if it is EmptyPacket
		if packet == EmptyPacket {
			return
		}
		resp, err := json.Marshal(packet)
		if err != nil {
			log.Println("[!] json marshal error:", err)
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, resp)
	}
}

// syncSend sends a packet and wait for callback till the packet comes back
func (c *Client) syncSend(request WSPacket) (response WSPacket) {
	res := make(chan WSPacket)
	f := func(resp WSPacket) {
		res <- resp
	}
	c.send(request, f)
	return <-res
}

func (c *Client) listen() {
	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("[!] read:", err)
			break
		}
		wspacket := WSPacket{}
		err = json.Unmarshal(rawMsg, &wspacket)
		if err != nil {
			continue
		}

		// Check if some async send is waiting for the response based on packetID
		if callback, ok := c.sendCallback[wspacket.PacketID]; ok {
			callback(wspacket)
			delete(c.sendCallback, wspacket.PacketID)
		}
		// Check if some receiver with the ID is registered
		if callback, ok := c.recvCallback[wspacket.ID]; ok {
			callback(wspacket)
		}
	}
}
