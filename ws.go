package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn           *websocket.Conn
	wsoverlord     *websocket.Conn
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

func NewClient(conn *websocket.Conn, webrtc *webrtc.WebRTC) *Client {
	sendCallback := map[string]func(WSPacket){}
	recvCallback := map[string]func(WSPacket){}
	return &Client{
		conn:           conn,
		peerconnection: webrtc,
		sendCallback:   sendCallback,
		recvCallback:   recvCallback,
	}
}

// syncSend sends a packet and trigger callback when the packet comes back
func (c *Client) syncSend(packet WSPacket, callback func(msg WSPacket)) {
	data, err := json.Marshal(packet)
	if err != nil {
		return
	}

	c.conn.WriteMessage(0, data)
	c.sendCallback[packet.PacketID] = callback
}

// syncReceive receive and response back
func (c *Client) syncReceive(id string, f func(request WSPacket) (response WSPacket)) {
	c.recvCallback[id] = func(request WSPacket) {
		packet := f(request)

		resp, err := json.Marshal(packet)
		if err != nil {
			log.Println("[!] json marshal error:", err)
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, resp)
	}
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
