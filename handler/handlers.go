package handler

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/cws"
	storage "github.com/giongto35/cloud-game/handler/cloud-storage"
	"github.com/giongto35/cloud-game/handler/gamelist"
	"github.com/giongto35/cloud-game/handler/room"
	"github.com/giongto35/cloud-game/webrtc"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	gameboyIndex = "./static/gameboy.html"
	debugIndex   = "./static/index_ws.html"
)

// Flag to determine if the server is overlord or not
var upgrader = websocket.Upgrader{}

type Handler struct {
	// Client that connects to overlord
	oClient *OverlordClient
	// Rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// ID of the current server globalwise
	serverID string
	// isDebug determines the mode handler is running
	isDebug bool
	// Path to game list
	gamePath string
	// All webrtc peerconnections are handled by the server
	// ID -> peerconnections
	peerconnections map[string]*webrtc.WebRTC
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage.Client
}

// NewHandler returns a new server
func NewHandler(overlordConn *websocket.Conn, isDebug bool, gamePath string) *Handler {
	onlineStorage := storage.NewInitClient()

	return &Handler{
		oClient:         NewOverlordClient(overlordConn),
		rooms:           map[string]*room.Room{},
		peerconnections: map[string]*webrtc.WebRTC{},

		isDebug:  isDebug,
		gamePath: gamePath,

		onlineStorage: onlineStorage,
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
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in f", r)
		}
	}()

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}
	defer c.Close()

	client := NewBrowserClient(c)
	sessionID := uuid.Must(uuid.NewV4()).String()
	wssession := &Session{
		ID:             sessionID,
		BrowserClient:  client,
		OverlordClient: h.oClient,
		peerconnection: webrtc.NewWebRTC(),
		handler:        h,
	}
	defer wssession.Close()

	if wssession.OverlordClient != nil {
		wssession.RouteOverlord()
		go wssession.OverlordClient.Heartbeat()
		go wssession.OverlordClient.Listen()
	}

	wssession.RouteBrowser()
	log.Println("oclient : ", h.oClient)

	wssession.BrowserClient.Send(cws.WSPacket{
		ID:   "gamelist",
		Data: gamelist.GetEncodedGameList(h.gamePath),
	}, nil)

	// If peerconnection is done (client.Done is signalled), we close peerconnection
	go func() {
		<-client.Done
		log.Println("Socket terminated, detach connection")
		h.detachPeerConn(wssession.peerconnection)
	}()

	wssession.BrowserClient.Listen()
}

// Detach peerconnection detach/remove a peerconnection from current room
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Println("Detach peerconnection")
	roomID := pc.RoomID
	room := h.getRoom(roomID)
	if room == nil {
		return
	}
	room.CleanSession(pc)
}

// getRoom returns room from roomID
func (h *Handler) getRoom(roomID string) *room.Room {
	room, ok := h.rooms[roomID]
	if !ok {
		return nil
	}

	return room
}

// detachRoom detach room from Handler
func (h *Handler) detachRoom(roomID string) {
	delete(h.rooms, roomID)
}

// createNewRoom creates a new room
// Return nil in case of room is existed
func (h *Handler) createNewRoom(gameName string, roomID string, playerIndex int) *room.Room {
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !h.isRoomRunning(roomID) {
		room := room.NewRoom(roomID, h.gamePath, gameName, h.onlineStorage)
		// TODO: Might have race condition
		h.rooms[room.ID] = room
		return room
	}

	return nil
}

// isRoomRunning check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed, we can check if the sessions list is empty or not.
func (h *Handler) isRoomRunning(roomID string) bool {
	// If no roomID is registered
	room, ok := h.rooms[roomID]
	if !ok {
		return false
	}

	return room.IsRunningSessions()
}
