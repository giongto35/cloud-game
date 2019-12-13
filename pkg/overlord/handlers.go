package overlord

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/cws"
	"github.com/giongto35/cloud-game/pkg/util"
	"github.com/giongto35/cloud-game/pkg/util/gamelist"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

const (
	gameboyIndex = "./web/game.html"
)

type Server struct {
	cfg Config
	// roomToWorker map roomID to workerID
	roomToWorker map[string]string
	// workerClients are the map workerID to worker Client
	workerClients map[string]*WorkerClient
}

var upgrader = websocket.Upgrader{}
var errNotFound = errors.New("Not found")

func NewServer(cfg Config) *Server {
	return &Server{
		cfg: cfg,
		// Mapping serverID to client
		workerClients: map[string]*WorkerClient{},
		// Mapping roomID to server
		roomToWorker: map[string]string{},
	}
}

type RenderData struct {
	STUNTURN string
}

// GetWeb returns web frontend
func (o *Server) GetWeb(w http.ResponseWriter, r *http.Request) {
	stunturn := *config.FrontendSTUNTURN
	if stunturn == "" {
		stunturn = config.DefaultSTUNTURN
	}
	data := RenderData{
		STUNTURN: stunturn,
	}

	tmpl, err := template.ParseFiles(gameboyIndex)
	if err != nil {
		log.Fatal(err)
	}

	tmpl.Execute(w, data)
}

// WSO handles all connections from a new worker to overlord
func (o *Server) WSO(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Connected")
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Overlord: [!] WS upgrade:", err)
		return
	}
	// Register new server
	serverID := uuid.Must(uuid.NewV4()).String()
	log.Println("Overlord: A new server connected to Overlord", serverID)

	// Register to workersClients map the client connection
	address := util.GetRemoteAddress(c)
	// Zone of the worker
	zone := r.URL.Query().Get("zone")

	fmt.Printf("Is public: %v zone: %v\n", util.IsPublicIP(address), zone)

	if !util.IsPublicIP(address) && *config.Mode == config.ProdEnv {
		// Don't accept private IP for worker's address in prod mode
		// However, if the worker in the same host with overlord, we can get public IP of worker
		log.Printf("Error: address %s is invalid", address)
		address = util.GetHostPublicIP()
		log.Println("Find public address:", address)
		if address == "" || !util.IsPublicIP(address) {
			// Skip this worker because we cannot find public IP
			return
		}
	}
	client := NewWorkerClient(c, serverID, address, fmt.Sprintf(config.StunTurnTemplate, address, address), zone)
	o.workerClients[serverID] = client
	defer o.cleanConnection(client, serverID)

	// Sendback the ID to server
	client.Send(
		cws.WSPacket{
			ID:   "serverID",
			Data: serverID,
		},
		nil,
	)
	o.RouteWorker(client)

	client.Listen()
}

// WSO handles all connections from user/frontend to overlord
func (o *Server) WS(w http.ResponseWriter, r *http.Request) {
	log.Println("A user connected to overlord ", r.URL)
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in ", r)
		}
	}()

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[!] WS upgrade:", err)
		return
	}
	defer c.Close()

	client := NewBrowserClient(c)
	go client.Listen()

	var workerClient *WorkerClient

	// get roomID if it is embeded in request. Server will pair the frontend with the server running the room. It only happens when we are trying to access a running room over share link.
	// TODO: Update link to the wiki
	roomID := r.URL.Query().Get("room_id")
	// zone param is to pick worker in that zone only
	// if there is no zone param, we can pic
	userZone := r.URL.Query().Get("zone")
	log.Println("A user connected to overlord ", roomID, userZone)

	log.Printf("Get Room %s Zone %s From URL %v", roomID, userZone, r.URL)

	if roomID != "" {
		log.Printf("Detected roomID %v from URL", roomID)
		if workerID, ok := o.roomToWorker[roomID]; ok {
			workerClient = o.workerClients[workerID]
			if userZone != "" && workerClient.Zone != userZone {
				// if there is zone param, we need to ensure ther worker in that zone
				// if not we consider the room is missing
				workerClient = nil
			} else {
				log.Printf("Found running server with id=%v client=%v", workerID, workerClient)
			}
		}
	}

	// If there is no existing server to connect to, we find the best possible worker for the frontend
	if workerClient == nil {
		// Get best server for frontend to connect to
		workerClient, err = o.getBestWorkerClient(client, userZone)
		if err != nil {
			return
		}
	}

	// SessionID will be the unique per frontend connection
	sessionID := uuid.Must(uuid.NewV4()).String()
	// Setup session
	wssession := &Session{
		ID:            sessionID,
		handler:       o,
		BrowserClient: client,
		WorkerClient:  workerClient,
		ServerID:      workerClient.ServerID,
	}
	// TODO:?
	// defer wssession.Close()
	log.Println("New client will conect to server", wssession.ServerID)
	wssession.WorkerClient.IsAvailable = false

	wssession.RouteBrowser()

	wssession.BrowserClient.Send(cws.WSPacket{
		ID:   "init",
		Data: createInitPackage(workerClient.StunTurnServer),
	}, nil)

	// If peerconnection is done (client.Done is signalled), we close peerconnection
	<-client.Done
	// Notify worker to clean session
	wssession.WorkerClient.Send(
		cws.WSPacket{
			ID:        "terminateSession",
			SessionID: sessionID,
		},
		nil,
	)
	// WorkerClient become available again
	wssession.WorkerClient.IsAvailable = true
}

func (o *Server) getBestWorkerClient(client *BrowserClient, zone string) (*WorkerClient, error) {
	if o.cfg.DebugHost != "" {
		log.Println("Connecting to debug host instead prod servers", o.cfg.DebugHost)
		wc := o.getWorkerFromAddress(o.cfg.DebugHost)
		if wc != nil {
			return wc, nil
		}
		// if there is not debugHost, continue usual flow
		log.Println("Not found, connecting to all available servers")
	}

	workerClients := o.getAvailableWorkers()
	log.Println("Current workerClients: ", o.workerClients)

	serverID, err := findBestServerFromBrowser(workerClients, client, zone)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return o.workerClients[serverID], nil
}

// getAvailableWorkers returns the list of available worker
func (o *Server) getAvailableWorkers() map[string]*WorkerClient {
	workerClients := map[string]*WorkerClient{}
	for k, w := range o.workerClients {
		if w.IsAvailable {
			workerClients[k] = w
		}
	}

	return workerClients
}

// getWorkerFromAddress returns the worker has given address
func (o *Server) getWorkerFromAddress(address string) *WorkerClient {
	for _, w := range o.workerClients {
		if w.IsAvailable && w.Address == address {
			return w
		}
	}

	return nil
}

// findBestServerFromBrowser returns the best server for a session
// All workers addresses are sent to user and user will ping to get latency
func findBestServerFromBrowser(workerClients map[string]*WorkerClient, client *BrowserClient, zone string) (string, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(workerClients) == 0 {
		return "", errors.New("No server found")
	}

	latencies := getLatencyMapFromBrowser(workerClients, client)
	log.Println("Latency map", latencies)

	if len(latencies) == 0 {
		return "", errors.New("No server found")
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

	return bestWorker.ServerID, nil
}

// getLatencyMapFromBrowser get all latencies from worker to user
func getLatencyMapFromBrowser(workerClients map[string]*WorkerClient, client *BrowserClient) map[*WorkerClient]int64 {
	workersList := []*WorkerClient{}

	latencyMap := map[*WorkerClient]int64{}

	// addressList is the list of worker addresses
	addressList := []string{}
	for _, workerClient := range workerClients {
		workersList = append(workersList, workerClient)
		addressList = append(addressList, workerClient.Address)
	}

	// send this address to user and get back latency
	log.Println("Send sync", addressList, strings.Join(addressList, ","))
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
		if latency, ok := respLatency[workerClient.Address]; ok {
			latencyMap[workerClient] = latency
		}
	}
	return latencyMap
}

// cleanConnection is called when a worker is disconnected
// connection from worker (client) to server is also closed
func (o *Server) cleanConnection(client *WorkerClient, serverID string) {
	log.Println("Unregister server from overlord")
	// Remove serverID from servers
	delete(o.workerClients, serverID)
	// Clean all rooms connecting to that server
	for roomID, roomServer := range o.roomToWorker {
		if roomServer == serverID {
			delete(o.roomToWorker, roomID)
		}
	}

	client.Close()
}

// createInitPackage returns serverhost + game list in encoded wspacket format
// This package will be sent to initialize
func createInitPackage(stunturn string) string {
	var gameName []string
	for _, game := range gamelist.GameList {
		gameName = append(gameName, game.Name)
	}

	initPackage := append([]string{stunturn}, gameName...)
	encodedList, _ := json.Marshal(initPackage)
	return string(encodedList)
}
