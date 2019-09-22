package overlord

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
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
	// roomToServer map roomID to workerID
	roomToServer map[string]string
	// workerClients are the map serverID to worker Client
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
		roomToServer: map[string]string{},
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
	fmt.Println("Is public", util.IsPublicIP(address))
	fmt.Println("Mode: ", *config.Mode, config.ProdEnv)
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
	client := NewWorkerClient(c, serverID, address, fmt.Sprintf(config.StunTurnTemplate, address, address))
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

// WSO handles all connections from frontend to overlord
func (o *Server) WS(w http.ResponseWriter, r *http.Request) {
	log.Println("Browser connected to overlord")
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

	// Get best server for frontend to connect to
	workerClient, err := o.getBestWorkerClient(client)
	if err != nil {
		return
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

func (o *Server) getBestWorkerClient(client *BrowserClient) (*WorkerClient, error) {
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

	var serverID string
	var err error
	if config.MatchWorkerRandom {
		serverID, err = findBestServerRandom(workerClients)
	} else {
		serverID, err = findBestServerFromBrowser(workerClients, client)
	}

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

// findBestServer returns the best server for a session
func findBestServerRandom(workerClients map[string]*WorkerClient) (string, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(workerClients) == 0 {
		return "", errors.New("No server found")
	}

	r := rand.Intn(len(workerClients))
	for k := range workerClients {
		if r == 0 {
			return k, nil
		}
		r--
	}

	return "", errors.New("No server found")
}

// findBestServerFromBrowser returns the best server for a session
// All workers addresses are sent to user and user will ping to get latency
func findBestServerFromBrowser(workerClients map[string]*WorkerClient, client *BrowserClient) (string, error) {
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
	for roomID, roomServer := range o.roomToServer {
		if roomServer == serverID {
			delete(o.roomToServer, roomID)
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
