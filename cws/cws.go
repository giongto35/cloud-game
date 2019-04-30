package cws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type Client struct {
	conn *websocket.Conn

	sendLock sync.Mutex
	// sendCallback is callback based on packetID
	sendCallback     map[string]func(req WSPacket)
	sendCallbackLock sync.Mutex
	// recvCallback is callback when receive based on ID of the packet
	recvCallback map[string]func(req WSPacket)
}

type WSPacket struct {
	ID   string `json:"id"`
	Data string `json:"data"`

	RoomID      string `json:"room_id"`
	PlayerIndex int    `json:"player_index"`

	TargetHostID string `json:"target_id"`
	PacketID     string `json:"packet_id"`
	// Globally ID of a session
	SessionID string `json:"session_id"`
}

var EmptyPacket = WSPacket{}

func NewClient(conn *websocket.Conn) *Client {
	sendCallback := map[string]func(WSPacket){}
	recvCallback := map[string]func(WSPacket){}
	return &Client{
		conn: conn,

		sendCallback: sendCallback,
		recvCallback: recvCallback,
	}
}

// Send sends a packet and trigger callback when the packet comes back
func (c *Client) Send(request WSPacket, callback func(response WSPacket)) {
	request.PacketID = uuid.Must(uuid.NewV4()).String()
	data, err := json.Marshal(request)
	if err != nil {
		return
	}

	// TODO: Consider using lock free
	c.sendLock.Lock()
	c.conn.WriteMessage(websocket.TextMessage, data)
	c.sendLock.Unlock()
	wrapperCallback := func(resp WSPacket) {
		resp.PacketID = request.PacketID
		resp.SessionID = request.SessionID
		callback(resp)
	}
	if callback == nil {
		return
	}
	c.sendCallbackLock.Lock()
	c.sendCallback[request.PacketID] = wrapperCallback
	c.sendCallbackLock.Unlock()
}

// Receive receive and response back
func (c *Client) Receive(id string, f func(response WSPacket) (request WSPacket)) {
	c.recvCallback[id] = func(response WSPacket) {
		req := f(response)
		// Add Meta data
		req.PacketID = response.PacketID
		req.SessionID = response.SessionID

		// Skip rqeuest if it is EmptyPacket
		if req == EmptyPacket {
			return
		}
		resp, err := json.Marshal(req)
		if err != nil {
			log.Println("[!] json marshal error:", err)
		}
		//c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.sendLock.Lock()
		c.conn.WriteMessage(websocket.TextMessage, resp)
		c.sendLock.Unlock()
	}
}

// SyncSend sends a packet and wait for callback till the packet comes back
func (c *Client) SyncSend(request WSPacket) (response WSPacket) {
	res := make(chan WSPacket)
	f := func(resp WSPacket) {
		res <- resp
	}
	c.Send(request, f)
	return <-res
}

// Heartbeat maintains connection to server
func (c *Client) Heartbeat() {
	// send heartbeat every 1s
	timer := time.Tick(time.Second)

	for range timer {
		c.Send(WSPacket{ID: "heartbeat"}, nil)
	}
}

func (c *Client) Listen() {
	for {
		//log.Println("Waiting for message ...")
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
		// TODO: Change to read lock
		c.sendCallbackLock.Lock()
		callback, ok := c.sendCallback[wspacket.PacketID]
		c.sendCallbackLock.Unlock()
		if ok {
			go callback(wspacket)
			c.sendCallbackLock.Lock()
			delete(c.sendCallback, wspacket.PacketID)
			c.sendCallbackLock.Unlock()
			// Skip receiveCallback to avoid duplication
			continue
		}
		// Check if some receiver with the ID is registered
		if callback, ok := c.recvCallback[wspacket.ID]; ok {
			go callback(wspacket)
		}
	}
}
