package coordinator

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ice"
	"github.com/giongto35/cloud-game/v2/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

const index = "./web/index.html"

type Server struct {
	cfg coordinator.Config
	// games library
	library games.GameLibrary
	// roomToWorker map roomID to workerID
	roomToWorker map[string]string
	// workerClients are the map workerID to worker Client
	workerClients map[string]*WorkerClient
	// browserClients are the map sessionID to browser Client
	browserClients map[string]*BrowserClient
}

const pingServerTemp = "https://%s.%s/echo"
const devPingServer = "http://localhost:9000/echo"

var upgrader = websocket.Upgrader{}

func NewServer(cfg coordinator.Config, library games.GameLibrary) *Server {
	return &Server{
		cfg:     cfg,
		library: library,
		// Mapping roomID to server
		roomToWorker: map[string]string{},
		// Mapping workerID to worker
		workerClients: map[string]*WorkerClient{},
		// Mapping sessionID to browser
		browserClients: map[string]*BrowserClient{},
	}
}

// GetWeb returns web frontend
func (s *Server) GetWeb(conf coordinator.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tpl, err := template.ParseFiles(index)
		if err != nil {
			log.Fatal(err)
		}

		// render index page with some tpl values
		if err = tpl.Execute(w, conf.Coordinator.Analytics); err != nil {
			log.Fatal(err)
		}
	})
}

// getPingServer returns the server for latency check of a zone.
// In latency check to find best worker step, we use this server to find the closest worker.
func (s *Server) getPingServer(zone string) string {
	if s.cfg.Coordinator.PingServer != "" {
		return fmt.Sprintf("%s/echo", s.cfg.Coordinator.PingServer)
	}

	mode := s.cfg.Environment.Get()
	if mode.AnyOf(environment.Production, environment.Staging) {
		return fmt.Sprintf(pingServerTemp, zone, s.cfg.Coordinator.PublicDomain)
	}
	return devPingServer
}

// WSO handles all connections from a new worker to coordinator
func (s *Server) WSO(w http.ResponseWriter, r *http.Request) {
	log.Println("Coordinator: A worker is connecting...")

	// be aware of ReadBufferSize, WriteBufferSize (default 4096)
	// https://pkg.go.dev/github.com/gorilla/websocket?tab=doc#Upgrader
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Coordinator: [!] WS upgrade:", err)
		return
	}

	// Generate workerID
	var workerID string
	for {
		workerID = uuid.Must(uuid.NewV4()).String()
		// check duplicate
		if _, ok := s.workerClients[workerID]; !ok {
			break
		}
	}

	// Create a workerClient instance
	wc := NewWorkerClient(c, workerID)
	wc.Println("Generated worker ID")

	// Register to workersClients map the client connection
	address := util.GetRemoteAddress(c)
	wc.Println("Address:", address)
	// Zone of the worker
	zone := r.URL.Query().Get("zone")
	wc.Printf("Is public: %v zone: %v", util.IsPublicIP(address), zone)

	pingServer := s.getPingServer(zone)

	wc.Printf("Set ping server address: %s", pingServer)

	// In case worker and coordinator in the same host
	if !util.IsPublicIP(address) && s.cfg.Environment.Get() == environment.Production {
		// Don't accept private IP for worker's address in prod mode
		// However, if the worker in the same host with coordinator, we can get public IP of worker
		wc.Printf("[!] Address %s is invalid", address)

		address = util.GetHostPublicIP()
		wc.Printf("Find public address: %s", address)

		if address == "" || !util.IsPublicIP(address) {
			// Skip this worker because we cannot find public IP
			wc.Println("[!] Unable to find public address, reject worker")
			return
		}
	}

	// Create a workerClient instance
	wc.Address = address
	wc.StunTurnServer = ice.ToJson(s.cfg.Webrtc.IceServers, ice.Replacement{From: "server-ip", To: address})
	wc.Zone = zone
	wc.PingServer = pingServer

	// Attach to Server instance with workerID, add defer
	s.workerClients[workerID] = wc
	defer s.cleanWorker(wc, workerID)

	wc.Send(api.ServerIdPacket(workerID), nil)

	s.workerRoutes(wc)
	wc.Listen()
}

// WSO handles all connections from user/frontend to coordinator
func (s *Server) WS(w http.ResponseWriter, r *http.Request) {
	log.Println("Coordinator: A user is connecting...")
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in ", r)
		}
	}()

	// be aware of ReadBufferSize, WriteBufferSize (default 4096)
	// https://pkg.go.dev/github.com/gorilla/websocket?tab=doc#Upgrader
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Coordinator: [!] WS upgrade:", err)
		return
	}

	// Generate sessionID for browserClient
	var sessionID string
	for {
		sessionID = uuid.Must(uuid.NewV4()).String()
		// check duplicate
		if _, ok := s.browserClients[sessionID]; !ok {
			break
		}
	}

	// Create browserClient instance
	bc := NewBrowserClient(c, sessionID)
	bc.Println("Generated worker ID")

	// Run browser listener first (to capture ping)
	go bc.Listen()

	/* Create a session - mapping browserClient with workerClient */
	var wc *WorkerClient

	// get roomID if it is embeded in request. Server will pair the frontend with the server running the room. It only happens when we are trying to access a running room over share link.
	// TODO: Update link to the wiki
	roomID := r.URL.Query().Get("room_id")
	// zone param is to pick worker in that zone only
	// if there is no zone param, we can pic
	userZone := r.URL.Query().Get("zone")

	bc.Printf("Get Room %s Zone %s From URL %v", roomID, userZone, r.URL)

	if roomID != "" {
		bc.Printf("Detected roomID %v from URL", roomID)
		if workerID, ok := s.roomToWorker[roomID]; ok {
			wc = s.workerClients[workerID]
			if userZone != "" && wc.Zone != userZone {
				// if there is zone param, we need to ensure ther worker in that zone
				// if not we consider the room is missing
				wc = nil
			} else {
				bc.Printf("Found running server with id=%v client=%v", workerID, wc)
			}
		}
	}

	// If there is no existing server to connect to, we find the best possible worker for the frontend
	if wc == nil {
		// Get best server for frontend to connect to
		wc, err = s.getBestWorkerClient(bc, userZone)
		if err != nil {
			return
		}
	}

	// Assign available worker to browserClient
	bc.WorkerID = wc.WorkerID

	wc.ChangeUserQuantityBy(1)
	defer wc.ChangeUserQuantityBy(-1)

	// Everything is cool
	// Attach to Server instance with sessionID
	s.browserClients[sessionID] = bc
	defer s.cleanBrowser(bc, sessionID)

	// Routing browserClient message
	s.useragentRoutes(bc)

	bc.Send(cws.WSPacket{
		ID:   "init",
		Data: createInitPackage(wc.StunTurnServer, s.library.GetAll()),
	}, nil)

	// If peerconnection is done (client.Done is signalled), we close peerconnection
	<-bc.Done

	// Notify worker to clean session
	wc.Send(api.TerminateSessionPacket(sessionID), nil)
}

func (s *Server) getBestWorkerClient(client *BrowserClient, zone string) (*WorkerClient, error) {
	conf := s.cfg.Coordinator
	if conf.DebugHost != "" {
		client.Println("Connecting to debug host instead prod servers", conf.DebugHost)
		wc := s.getWorkerFromAddress(conf.DebugHost)
		if wc != nil {
			return wc, nil
		}
		// if there is not debugHost, continue usual flow
		client.Println("Not found, connecting to all available servers")
	}

	workerClients := s.getAvailableWorkers()

	serverID, err := s.findBestServerFromBrowser(workerClients, client, zone)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return s.workerClients[serverID], nil
}

// getAvailableWorkers returns the list of available worker
func (s *Server) getAvailableWorkers() map[string]*WorkerClient {
	workerClients := map[string]*WorkerClient{}
	for k, w := range s.workerClients {
		if w.HasGameSlot() {
			workerClients[k] = w
		}
	}

	return workerClients
}

// getWorkerFromAddress returns the worker has given address
func (s *Server) getWorkerFromAddress(address string) *WorkerClient {
	for _, w := range s.workerClients {
		if w.HasGameSlot() && w.Address == address {
			return w
		}
	}

	return nil
}

// findBestServerFromBrowser returns the best server for a session
// All workers addresses are sent to user and user will ping to get latency
func (s *Server) findBestServerFromBrowser(workerClients map[string]*WorkerClient, client *BrowserClient, zone string) (string, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(workerClients) == 0 {
		return "", errors.New("no server found")
	}

	latencies := s.getLatencyMapFromBrowser(workerClients, client)
	client.Println("Latency map", latencies)

	if len(latencies) == 0 {
		return "", errors.New("no server found")
	}

	var bestWorker *WorkerClient
	var minLatency int64 = math.MaxInt64

	// get the worker with lowest latency to user
	for wc, l := range latencies {
		if zone != "" && wc.Zone != zone {
			// skip worker not in the zone if zone param is given
			continue
		}

		if l < minLatency {
			bestWorker = wc
			minLatency = l
		}
	}

	return bestWorker.WorkerID, nil
}

// getLatencyMapFromBrowser get all latencies from worker to user
func (s *Server) getLatencyMapFromBrowser(workerClients map[string]*WorkerClient, client *BrowserClient) map[*WorkerClient]int64 {
	var workersList []*WorkerClient
	var addressList []string
	uniqueAddresses := map[string]bool{}
	latencyMap := map[*WorkerClient]int64{}

	// addressList is the list of worker addresses
	for _, workerClient := range workerClients {
		if _, ok := uniqueAddresses[workerClient.PingServer]; !ok {
			addressList = append(addressList, workerClient.PingServer)
		}
		uniqueAddresses[workerClient.PingServer] = true
		workersList = append(workersList, workerClient)
	}

	// send this address to user and get back latency
	client.Println("Send sync", addressList, strings.Join(addressList, ","))
	data := client.SyncSend(cws.WSPacket{
		ID:   "checkLatency",
		Data: strings.Join(addressList, ","),
	})

	respLatency := map[string]int64{}
	err := json.Unmarshal([]byte(data.Data), &respLatency)
	if err != nil {
		log.Println(err)
		return latencyMap
	}

	for _, workerClient := range workersList {
		if latency, ok := respLatency[workerClient.PingServer]; ok {
			latencyMap[workerClient] = latency
		}
	}
	return latencyMap
}

// cleanBrowser is called when a browser is disconnected
func (s *Server) cleanBrowser(bc *BrowserClient, sessionID string) {
	bc.Println("Disconnect from coordinator")
	delete(s.browserClients, sessionID)
	bc.Close()
}

// cleanWorker is called when a worker is disconnected
// connection from worker to coordinator is also closed
func (s *Server) cleanWorker(wc *WorkerClient, workerID string) {
	wc.Println("Unregister worker from coordinator")
	// Remove workerID from workerClients
	delete(s.workerClients, workerID)
	// Clean all rooms connecting to that server
	for roomID, roomServer := range s.roomToWorker {
		if roomServer == workerID {
			wc.Printf("Remove room %s", roomID)
			delete(s.roomToWorker, roomID)
		}
	}

	wc.Close()
}

// createInitPackage returns serverhost + game list in encoded wspacket format
// This package will be sent to initialize
func createInitPackage(stunturn string, games []games.GameMetadata) string {
	var gameName []string
	for _, game := range games {
		gameName = append(gameName, game.Name)
	}

	initPackage := append([]string{stunturn}, gameName...)
	encodedList, _ := json.Marshal(initPackage)
	return string(encodedList)
}
