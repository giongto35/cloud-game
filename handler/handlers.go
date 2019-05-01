package handler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/handler/gamelist"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	scale        = 3
	title        = "NES"
	gameboyIndex = "./static/gameboy.html"
	debugIndex   = "./static/index_ws.html"
)

// Time allowed to write a message to the peer.
var readWait = 30 * time.Second
var writeWait = 30 * time.Second

// Flag to determine if the server is overlord or not
var upgrader = websocket.Upgrader{}

// ID to peerconnection
//var peerconnections = map[string]*webrtc.WebRTC{}
//var oclient *OverlordClient

type Handler struct {
	oClient  *OverlordClient
	rooms    map[string]*Room
	serverID string
	// isDebug determines the mode handler is running
	isDebug    bool
	isOverlord bool

	// ID to peerconnection
	peerconnections map[string]*webrtc.WebRTC
	// Session
	wssession Session
}

// NewHandler returns a new server
func NewHandler(overlordConn *websocket.Conn, isDebug bool) *Handler {
	//conn, err := createOverlordConnection()
	//if err != nil {
	//return nil, err
	//}
	return &Handler{
		isDebug:         isDebug,
		oClient:         NewOverlordClient(overlordConn),
		rooms:           map[string]*Room{},
		peerconnections: map[string]*webrtc.WebRTC{},
	}
}

// GetWeb returns web frontend
func (h *Handler) GetWeb(w http.ResponseWriter, r *http.Request) {
	indexFN := ""
	if h.isDebug {
		indexFN = debugIndex
	} else {
		indexFN = gameboyIndex
	}

	bs, err := ioutil.ReadFile(indexFN)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bs)
}

// WS handles normal traffic (from browser to host)
func (h *Handler) WS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}

	client := NewBrowserClient(c)
	//client := NewClient(c)
	////sessionID := strconv.Itoa(rand.Int())
	sessionID := uuid.Must(uuid.NewV4()).String()
	wssession := &Session{
		ID:             sessionID,
		BrowserClient:  client,
		OverlordClient: h.oClient,
		peerconnection: webrtc.NewWebRTC(),
	}
	wssession.RegisterBrowserClient()
	fmt.Println("oclient : ", h.oClient)

	if wssession.OverlordClient != nil {
		wssession.RegisterOverlordClient()
		go wssession.OverlordClient.Heartbeat()
		go wssession.OverlordClient.Listen()
	}

	wssession.BrowserClient.Send(cws.WSPacket{
		ID:   "gamelist",
		Data: gamelist.GetEncodedGameList(),
	}, nil)

	wssession.BrowserClient.Listen()

	defer c.Close()
	//var gameName string
	//var roomID string
	//var playerIndex int

	//// Create connection to overlord
	//client := NewClient(c)
	////sessionID := strconv.Itoa(rand.Int())
	//sessionID := uuid.Must(uuid.NewV4()).String()

	//wssession := &Session{
	//client:         client,
	//peerconnection: webrtc.NewWebRTC(),
	//// The server session is maintaining
	//}

	//client.send(WSPacket{
	//ID:   "gamelist",
	//Data: getEncodedGameList(),
	//}, nil)

	//client.receive("heartbeat", func(resp WSPacket) WSPacket {
	//return resp
	//})

	//client.receive("initwebrtc", func(resp WSPacket) WSPacket {
	//log.Println("Received user SDP")
	//localSession, err := wssession.peerconnection.StartClient(resp.Data, width, height)
	//if err != nil {
	//log.Fatalln(err)
	//}

	//return WSPacket{
	//ID:        "sdp",
	//Data:      localSession,
	//SessionID: sessionID,
	//}
	//})

	//client.receive("save", func(resp WSPacket) (req WSPacket) {
	//log.Println("Saving game state")
	//req.ID = "save"
	//req.Data = "ok"
	//if roomID != "" {
	//err = rooms[roomID].director.SaveGame()
	//if err != nil {
	//log.Println("[!] Cannot save game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	//client.receive("load", func(resp WSPacket) (req WSPacket) {
	//log.Println("Loading game state")
	//req.ID = "load"
	//req.Data = "ok"
	//if roomID != "" {
	//err = rooms[roomID].director.LoadGame()
	//if err != nil {
	//log.Println("[!] Cannot load game state: ", err)
	//req.Data = "error"
	//}
	//} else {
	//req.Data = "error"
	//}

	//return req
	//})

	//client.receive("start", func(resp WSPacket) (req WSPacket) {
	//gameName = resp.Data
	//roomID = resp.RoomID
	//playerIndex = resp.PlayerIndex
	//isNewRoom := false

	//log.Println("Starting game")
	//// If we are connecting to overlord, request serverID from roomID
	//if oclient != nil {
	//roomServerID := getServerIDOfRoom(oclient, roomID)
	//log.Println("Server of RoomID ", roomID, " is ", roomServerID)
	//if roomServerID != "" && wssession.ServerID != roomServerID {
	//// TODO: Re -register
	//go bridgeConnection(wssession, roomServerID, gameName, roomID, playerIndex)
	//return
	//}
	//}

	//roomID, isNewRoom = startSession(wssession.peerconnection, gameName, roomID, playerIndex)
	//// Register room to overlord if we are connecting to overlord
	//if isNewRoom && oclient != nil {
	//oclient.send(WSPacket{
	//ID:   "registerRoom",
	//Data: roomID,
	//}, nil)
	//}
	//req.ID = "start"
	//req.RoomID = roomID
	//req.SessionID = sessionID

	//return req
	//})

	//client.receive("candidate", func(resp WSPacket) (req WSPacket) {
	//// Unuse code
	//hi := pionRTC.ICECandidateInit{}
	//err = json.Unmarshal([]byte(resp.Data), &hi)
	//if err != nil {
	//log.Println("[!] Cannot parse candidate: ", err)
	//} else {
	//// webRTC.AddCandidate(hi)
	//}
	//req.ID = "candidate"

	//return req
	//})

	//client.Listen()
}

func createOverlordConnection() (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(*config.OverlordHost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}
