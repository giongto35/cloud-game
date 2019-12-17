package worker

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/giongto35/cloud-game/pkg/config/worker"

	"github.com/giongto35/cloud-game/pkg/util"
	"github.com/giongto35/cloud-game/pkg/webrtc"
	storage "github.com/giongto35/cloud-game/pkg/worker/cloud-storage"
	"github.com/giongto35/cloud-game/pkg/worker/room"
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
	cfg          worker.Config
	// Rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// ID of the current server globalwise
	serverID string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage.Client
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[string]*Session
}

// NewHandler returns a new server
func NewHandler(cfg worker.Config) *Handler {
	// Create offline storage folder
	createOfflineStorage()

	// Init online storage
	onlineStorage := storage.NewInitClient()
	return &Handler{
		rooms:         map[string]*room.Room{},
		sessions:      map[string]*Session{},
		overlordHost:  cfg.OverlordAddress,
		cfg:           cfg,
		onlineStorage: onlineStorage,
	}
}

// Run starts a Handler running logic
func (h *Handler) Run() {
	for {
		oClient, err := setupOverlordConnection(h.overlordHost, h.cfg.Zone)
		if err != nil {
			log.Printf("Cannot connect to overlord. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}

		h.oClient = oClient
		log.Println("Connected to overlord successfully.", oClient, err)
		go h.oClient.Heartbeat()
		h.RouteOverlord()
		h.oClient.Listen()
		// If cannot listen, reconnect to overlord
	}
}

func setupOverlordConnection(ohost string, zone string) (*OverlordClient, error) {
	overlordURL := url.URL{
		Scheme:   "wss",
		Host:     ohost,
		Path:     "/wso",
		RawQuery: "zone=" + zone,
	}
	log.Println("Worker connecting to overlord:", overlordURL.String())

	conn, err := createOverlordConnection(&overlordURL)
	if err != nil {
		return nil, err
	}
	return NewOverlordClient(conn), nil
}

func createOverlordConnection(ourl *url.URL) (*websocket.Conn, error) {

	roots := x509.NewCertPool()
	severCert, err := ioutil.ReadFile("./server.crt")
	if err != nil {
		fmt.Println("Could not load server certificate!")
		log.Fatal(err)
	}
	if ok := roots.AppendCertsFromPEM(severCert); !ok {
		log.Fatal("Cannot append serverCert to Root")
	}

	d := websocket.Dialer{TLSClientConfig: &tls.Config{RootCAs: roots, InsecureSkipVerify: true}}
	ws, _, err := d.Dial(ourl.String(), nil)
	if err != nil {
		return nil, err
	}

	return ws, nil
}

func (h *Handler) GetOverlordClient() *OverlordClient {
	return h.oClient
}

// detachPeerConn detach/remove a peerconnection from current room
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Println("Detach peerconnection")
	room := h.getRoom(pc.RoomID)
	if room == nil {
		return
	}

	if !room.EmptySessions() {
		room.RemoveSession(pc)
		// If no more session in that room, we close that room
		if room.EmptySessions() {
			log.Println("No session in room")
			room.Close()
			// Signal end of input Channel
			log.Println("Signal input chan")
			pc.InputChannel <- -1
			close(pc.InputChannel)
		}
	}
}

// getRoom returns room from roomID
func (h *Handler) getRoom(roomID string) *room.Room {
	room, ok := h.rooms[roomID]
	if !ok {
		return nil
	}

	return room
}

// getRoom returns session from sessionID
func (h *Handler) getSession(sessionID string) *Session {
	session, ok := h.sessions[sessionID]
	if !ok {
		return nil
	}

	return session
}

// detachRoom detach room from Handler
func (h *Handler) detachRoom(roomID string) {
	delete(h.rooms, roomID)
}

// createNewRoom creates a new room
// Return nil in case of room is existed
func (h *Handler) createNewRoom(gameName string, roomID string, videoEncoderType string) *room.Room {
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !h.isRoomRunning(roomID) {
		room := room.NewRoom(roomID, gameName, videoEncoderType, h.onlineStorage, h.cfg)
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
func createOfflineStorage() {
	dir, _ := path.Split(util.GetSavePath("dummy"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("Failed to create offline storage, err: ", err)
	}
}
