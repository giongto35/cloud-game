package worker

import (
	"crypto/tls"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	storage "github.com/giongto35/cloud-game/v2/pkg/worker/cloud-storage"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
	"github.com/gorilla/websocket"
)

type Handler struct {
	// Client that connects to coordinator
	oClient *CoordinatorClient
	// Raw address of coordinator
	coordinatorHost string
	cfg             worker.Config
	// Rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// global ID of the current server
	serverID string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage.Client
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[string]*Session

	w *Worker
}

// NewHandler returns a new server
func NewHandler(cfg worker.Config, wrk *Worker) *Handler {
	// Create offline storage folder
	createOfflineStorage(cfg.Emulator.Storage)

	// Init online storage
	onlineStorage := storage.NewInitClient()
	return &Handler{
		rooms:           map[string]*room.Room{},
		sessions:        map[string]*Session{},
		coordinatorHost: cfg.Worker.Network.CoordinatorAddress,
		cfg:             cfg,
		onlineStorage:   onlineStorage,
		w:               wrk,
	}
}

// Run starts a Handler running logic
func (h *Handler) Run() {
	conf := h.cfg.Worker.Network
	for {
		conn, err := setupCoordinatorConnection(conf.CoordinatorAddress, conf.Zone, h.cfg)
		if err != nil {
			log.Printf("Cannot connect to coordinator. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[worker] connected to: %v", conf.CoordinatorAddress)

		h.oClient = conn
		go h.oClient.Heartbeat()
		h.routes()
		h.oClient.Listen()
		// If cannot listen, reconnect to coordinator
	}
}

func (h *Handler) RequestConfig() {
	log.Printf("[worker] asking for a config...")
	response := h.oClient.SyncSend(api.ConfigPacket())
	conf := worker.EmptyConfig()
	conf.Deserialize([]byte(response.Data))
	log.Printf("[worker] pulled config: %+v", conf)
}

func (h *Handler) Prepare() {
	if !h.cfg.Emulator.Libretro.Cores.Repo.Sync {
		return
	}

	log.Printf("Starting Libretro cores sync...")
	coreManager := remotehttp.NewRemoteHttpManager(h.cfg.Emulator.Libretro)
	if err := coreManager.Sync(); err != nil {
		log.Printf("error: cores sync has failed, %v", err)
	}
}

func setupCoordinatorConnection(host string, zone string, cfg worker.Config) (*CoordinatorClient, error) {
	var scheme string
	env := cfg.Environment.Get()
	if env.AnyOf(environment.Production, environment.Staging) {
		scheme = "wss"
	} else {
		scheme = "ws"
	}

	coordinatorURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     "/wso",
		RawQuery: "zone=" + zone,
	}
	log.Println("Worker connecting to coordinator:", coordinatorURL.String())

	conn, err := createCoordinatorConnection(&coordinatorURL)
	if err != nil {
		return nil, err
	}
	return NewCoordinatorClient(conn), nil
}

func createCoordinatorConnection(url *url.URL) (*websocket.Conn, error) {
	var d websocket.Dialer
	if url.Scheme == "wss" {
		d = websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	} else {
		d = websocket.Dialer{}
	}

	ws, _, err := d.Dial(url.String(), nil)
	if err != nil {
		return nil, err
	}

	return ws, nil
}

func (h *Handler) GetCoordinatorClient() *CoordinatorClient {
	return h.oClient
}

// detachPeerConn detaches a peerconnection from the current room.
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Printf("[worker] closing peer connection")
	gameRoom := h.getRoom(pc.RoomID)
	if gameRoom == nil || gameRoom.IsEmpty() {
		return
	}
	gameRoom.RemoveSession(pc)
	if gameRoom.IsEmpty() {
		log.Printf("[worker] closing an empty room")
		gameRoom.Close()
		pc.InputChannel <- []byte{0xFF, 0xFF}
		close(pc.InputChannel)
	}
}

func (h *Handler) getRoom(roomID string) (r *room.Room) {
	r, ok := h.rooms[roomID]
	if !ok {
		return nil
	}
	return
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
func (h *Handler) createNewRoom(game games.GameMetadata, roomID string, videoCodec encoder.VideoCodec) *room.Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if !h.isRoomBusy(roomID) {
		newRoom := room.NewRoom(roomID, game, videoCodec, h.onlineStorage, h.cfg)
		// TODO: Might have race condition (and it has (:)
		h.rooms[newRoom.ID] = newRoom
		return newRoom
	}
	return nil
}

// isRoomBusy check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed,
// we can check if the sessions list is empty or not.
func (h *Handler) isRoomBusy(roomID string) bool {
	if roomID == "" {
		return false
	}
	// If no roomID is registered
	r, ok := h.rooms[roomID]
	if !ok {
		return false
	}
	return r.IsRunningSessions()
}

func (h *Handler) Close() {
	if h.oClient != nil {
		h.oClient.Close()
	}
	for _, r := range h.rooms {
		r.Close()
	}
}

func createOfflineStorage(storage string) {
	dir, _ := path.Split(storage)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("Failed to create offline storage, err: ", err)
	}
}
