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
	id string

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
	id := uuid.Must(uuid.NewV4()).String()
	sendCallback := map[string]func(WSPacket){}
	recvCallback := map[string]func(WSPacket){}

	return &Client{
		id:   id,
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
	// Wrap callback with sessionID and packetID
	if callback != nil {
		wrapperCallback := func(resp WSPacket) {
			resp.PacketID = request.PacketID
			resp.SessionID = request.SessionID
			callback(resp)
		}
		c.sendCallbackLock.Lock()
		c.sendCallback[request.PacketID] = wrapperCallback
		c.sendCallbackLock.Unlock()
	}
	//log.Println("Registered requested callback", "ID :", request.ID, "PacketID: ", request.PacketID)
	//log.Println("Callback waiting list:", c.id, c.sendCallback)

	c.sendLock.Lock()
	c.conn.WriteMessage(websocket.TextMessage, data)
	c.sendLock.Unlock()
}

// Receive receive and response back
func (c *Client) Receive(id string, f func(response WSPacket) (request WSPacket)) {
	c.recvCallback[id] = func(response WSPacket) {
		req := f(response)
		// Add Meta data
		req.PacketID = response.PacketID
		req.SessionID = response.SessionID
		//log.Println("Sending back request", req, "PacketID: ", req.PacketID, "SessionID: ", req.SessionID)

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
		//log.Println( "Received: ", wspacket)

		if err != nil {
			continue
		}

		// Check if some async send is waiting for the response based on packetID
		// TODO: Change to read lock
		c.sendCallbackLock.Lock()
		//log.Println("Listening: Callback waiting list: ", c.id, c.sendCallback)
		callback, ok := c.sendCallback[wspacket.PacketID]
		//log.Println("Has callback: ", ok, "ClientID: ", c.id, "PacketID ", wspacket.PacketID)
		c.sendCallbackLock.Unlock()
		if ok {
			go callback(wspacket)
			c.sendCallbackLock.Lock()
			//log.Println("Deleteing Packet ", wspacket.PacketID)
			delete(c.sendCallback, wspacket.PacketID)
			c.sendCallbackLock.Unlock()
			// Skip receiveCallback to avoid duplication
			continue
		}
		//log.Println("Listening: Callback waiting list: ", c.id, c.recvCallback)
		// Check if some receiver with the ID is registered
		if callback, ok := c.recvCallback[wspacket.ID]; ok {
			go callback(wspacket)
		}
	}
}

func (c *Client) Close() {
	c.conn.Close()
}
