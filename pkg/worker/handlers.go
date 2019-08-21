package worker

import (
	"log"
	"time"

	"github.com/giongto35/cloud-game/pkg/webrtc"
	storage2 "github.com/giongto35/cloud-game/pkg/worker/cloud-storage"
	room2 "github.com/giongto35/cloud-game/pkg/worker/room"
	"github.com/gorilla/websocket"
)

const (
	gameboyIndex = "./static/game.html"
	debugIndex   = "./static/game.html"
)

// Flag to determine if the server is overlord or not
var upgrader = websocket.Upgrader{}

type Handler struct {
	// Client that connects to overlord
	oClient *OverlordClient
	// Raw address of overlord
	overlordHost string
	// Rooms map : RoomID -> Room
	rooms map[string]*room2.Room
	// ID of the current server globalwise
	serverID string
	// Path to game list
	gamePath string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage2.Client
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[string]*Session
}

// NewHandler returns a new server
func NewHandler(overlordHost string, gamePath string) *Handler {
	onlineStorage := storage2.NewInitClient()

	return &Handler{
		rooms:         map[string]*room2.Room{},
		sessions:      map[string]*Session{},
		overlordHost:  overlordHost,
		gamePath:      gamePath,
		onlineStorage: onlineStorage,
	}
}

// Run starts a Handler running logic
func (h *Handler) Run() {
	for {
		oClient, err := setupOverlordConnection(h.overlordHost)
		if err != nil {
			log.Println("Cannot connect to overlord. Retrying...")
			time.Sleep(time.Second)
			continue
		}

		h.oClient = oClient
		log.Println("Connected to overlord successfully.")
		go h.oClient.Heartbeat()
		h.RouteOverlord()
		h.oClient.Listen()
		// If cannot listen, reconnect to overlord
	}
}

func setupOverlordConnection(ohost string) (*OverlordClient, error) {
	conn, err := createOverlordConnection(ohost)
	if err != nil {
		return nil, err
	}
	return NewOverlordClient(conn), nil
}

func createOverlordConnection(ohost string) (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(ohost, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (h *Handler) GetOverlordClient() *OverlordClient {
	return h.oClient
}

// detachPeerConn detach/remove a peerconnection from current room
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Println("Detach peerconnection")
	roomID := pc.RoomID
	room := h.getRoom(roomID)
	if room == nil {
		return
	}

	// If room has no sessions, close room
	if !room.EmptySessions() {
		room.RemoveSession(pc)
		if room.EmptySessions() {
			log.Println("No session in room")
			room.Close()
			// Signal end of input Channel
			log.Println("Signal input chan")
			pc.InputChannel <- -1
		}
	}
}

// getRoom returns room from roomID
func (h *Handler) getRoom(roomID string) *room2.Room {
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
func (h *Handler) createNewRoom(gameName string, roomID string, playerIndex int) *room2.Room {
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !h.isRoomRunning(roomID) {
		room := room2.NewRoom(roomID, h.gamePath, gameName, h.onlineStorage)
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

func (h *Handler) Close() {
	if h.oClient != nil {
		h.oClient.Close()
	}
	// Close all room
	for _, room := range h.rooms {
		room.Close()
	}
}
