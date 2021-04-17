package coordinator

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"html/template"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ice"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	ws "github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

type Server struct {
	cfg coordinator.Config
	// games library
	library games.GameLibrary
	// roomToWorker map roomID to workerID
	roomToWorker map[string]network.Uid
	// workerClients are the map workerID to worker Client
	workerClients map[network.Uid]*WorkerClient
	// browserClients are the map sessionID to browser Client
	browserClients map[network.Uid]*BrowserClient
}

const pingServerTemp = "https://%s.%s/echo"
const devPingServer = "http://localhost:9000/echo"

func NewServer(cfg coordinator.Config, library games.GameLibrary) *Server {
	return &Server{
		cfg:     cfg,
		library: library,
		// Mapping roomID to server
		roomToWorker: map[string]network.Uid{},
		// Mapping workerID to worker
		workerClients: map[network.Uid]*WorkerClient{},
		// Mapping sessionID to browser
		browserClients: map[network.Uid]*BrowserClient{},
	}
}

func (c *Server) RelayPacket(u *BrowserClient, packet cws.WSPacket, req func(w *WorkerClient, p cws.WSPacket) cws.WSPacket) cws.WSPacket {
	packet.SessionID = u.SessionID
	wc, ok := c.workerClients[u.Worker.WorkerID]
	if !ok {
		return cws.EmptyPacket
	}
	return req(wc, packet)
}

func index(conf coordinator.Config) http.Handler {
	tpl, err := template.ParseFiles("./web/index.html")
	if err != nil {
		log.Fatal(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// return 404 on unknown
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// render index page with some tpl values
		if err = tpl.Execute(w, conf.Coordinator.Analytics); err != nil {
			log.Fatal(err)
		}
	})
}

func static(dir string) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))
}

// getPingServer returns the server for latency check of a zone.
// In latency check to find best worker step, we use this server to find the closest worker.
func (c *Server) getPingServer(zone string) string {
	if c.cfg.Coordinator.PingServer != "" {
		return fmt.Sprintf("%s/echo", c.cfg.Coordinator.PingServer)
	}

	if c.cfg.Coordinator.Server.Https && c.cfg.Coordinator.Server.Tls.Domain != "" {
		return fmt.Sprintf(pingServerTemp, zone, c.cfg.Coordinator.Server.Tls.Domain)
	}
	return devPingServer
}

// WSO handles all connections from a new worker to coordinator
func (c *Server) WSO(w http.ResponseWriter, r *http.Request) {
	log.Printf("New worker connection...")

	conn, err := ws.Upgrade(w, r)
	if err != nil {
		log.Printf("error: socket upgrade failed because of %v", err)
		return
	}

	// Generate workerID
	workerID := network.NewUid()

	// Create a workerClient instance
	wc := NewWorkerClient(conn, workerID)
	wc.Println("Generated worker ID")

	// Register to workersClients map the client connection
	address := util.GetRemoteAddress(conn)
	wc.Println("Address:", address)
	// Zone of the worker
	zone := r.URL.Query().Get("zone")
	wc.Printf("Is public: %v zone: %v", util.IsPublicIP(address), zone)

	pingServer := c.getPingServer(zone)

	wc.Printf("Set ping server address: %s", pingServer)

	// In case worker and coordinator in the same host
	if !util.IsPublicIP(address) && c.cfg.Environment.Get() == environment.Production {
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
	wc.StunTurnServer = ice.ToJson(c.cfg.Webrtc.IceServers, ice.Replacement{From: "server-ip", To: address})
	wc.Zone = zone
	wc.PingServer = pingServer

	// Attach to Server instance with workerID, add defer
	c.workerClients[workerID] = wc
	defer c.cleanWorker(wc, workerID)

	wc.Send(api.ServerIdPacket(workerID), nil)

	c.workerRoutes(wc)
	wc.Listen()
}

// WS handles all connections from user/frontend to coordinator
func (c *Server) WS(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in ", r)
		}
	}()

	conn, err := ws.Upgrade(w, r)
	if err != nil {
		log.Printf("error: socket upgrade failed because of %v", err)
		return
	}

	user := NewBrowserClient(conn, network.NewUid())

	// Run browser listener first (to capture ping)
	go user.Listen()

	/* Create a session - mapping browserClient with workerClient */
	var worker *WorkerClient

	// get roomID if it is embeded in request. Server will pair the frontend with the server running the room. It only happens when we are trying to access a running room over share link.
	// TODO: Update link to the wiki
	roomID := r.URL.Query().Get("room_id")
	// zone param is to pick worker in that zone only
	// if there is no zone param, we can pic
	userZone := r.URL.Query().Get("zone")

	user.Printf("Get Room %s Zone %s From URL %v", roomID, userZone, r.URL)

	if roomID != "" {
		user.Printf("Detected roomID %v from URL", roomID)
		if workerID, ok := c.roomToWorker[roomID]; ok {
			worker = c.workerClients[workerID]
			if userZone != "" && worker.Zone != userZone {
				// if there is zone param, we need to ensure ther worker in that zone
				// if not we consider the room is missing
				worker = nil
			} else {
				user.Printf("Found running server with id=%v client=%v", workerID, worker)
			}
		}
	}

	// If there is no existing server to connect to, we find the best possible worker for the frontend
	if worker == nil {
		// Get best server for frontend to connect to
		worker, err = c.getBestWorkerClient(user, userZone)
		if err != nil {
			return
		}
	}

	user.AssignWorker(worker)

	// Attach to Server instance with sessionID
	c.browserClients[user.SessionID] = user
	defer c.cleanBrowser(user)
	// Routing browserClient message
	c.useragentRoutes(user)

	user.SendPacket(api.InitPacket(createInitPackage(worker.StunTurnServer, c.library.GetAll())))

	// If peerconnection is done (client.Done is signalled), we close peerconnection
	<-user.Done

	// Notify worker to clean session
	worker.SendPacket(api.TerminateSessionPacket(user.SessionID))
	user.RetainWorker()
}

func (c *Server) getBestWorkerClient(client *BrowserClient, zone string) (*WorkerClient, error) {
	conf := c.cfg.Coordinator
	if conf.DebugHost != "" {
		client.Println("Connecting to debug host instead prod servers", conf.DebugHost)
		wc := c.getWorkerFromAddress(conf.DebugHost)
		if wc != nil {
			return wc, nil
		}
		// if there is not debugHost, continue usual flow
		client.Println("Not found, connecting to all available servers")
	}

	workerClients := c.getAvailableWorkers()

	serverID, err := c.findBestServerFromBrowser(workerClients, client, zone)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return c.workerClients[serverID], nil
}

// getAvailableWorkers returns the list of available worker
func (c *Server) getAvailableWorkers() map[network.Uid]*WorkerClient {
	workerClients := map[network.Uid]*WorkerClient{}
	for k, w := range c.workerClients {
		if w.IsAvailable {
			workerClients[k] = w
		}
	}
	return workerClients
}

// getWorkerFromAddress returns the worker has given address
func (c *Server) getWorkerFromAddress(address string) *WorkerClient {
	for _, w := range c.workerClients {
		if w.IsAvailable && w.Address == address {
			return w
		}
	}

	return nil
}

// findBestServerFromBrowser returns the best server for a session
// All workers addresses are sent to user and user will ping to get latency
func (c *Server) findBestServerFromBrowser(workerClients map[network.Uid]*WorkerClient, client *BrowserClient, zone string) (network.Uid, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(workerClients) == 0 {
		return [16]byte{}, errors.New("no server found")
	}

	latencies := c.getLatencyMapFromBrowser(workerClients, client)
	client.Println("Latency map", latencies)

	if len(latencies) == 0 {
		return [16]byte{}, errors.New("no server found")
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

	if bestWorker == nil {
		return [16]byte{}, errors.New("no server found")
	}

	return bestWorker.WorkerID, nil
}

// getLatencyMapFromBrowser get all latencies from worker to user
func (c *Server) getLatencyMapFromBrowser(workerClients map[network.Uid]*WorkerClient, client *BrowserClient) map[*WorkerClient]int64 {
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
	data := client.SyncSend(api.CheckLatencyPacket(addressList))

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
func (c *Server) cleanBrowser(bc *BrowserClient) {
	bc.Println("Disconnect from coordinator")
	delete(c.browserClients, bc.SessionID)
	bc.Close()
}

// cleanWorker is called when a worker is disconnected
// connection from worker to coordinator is also closed
func (c *Server) cleanWorker(wc *WorkerClient, workerID network.Uid) {
	wc.Println("Unregister worker from coordinator")
	// Remove workerID from workerClients
	delete(c.workerClients, workerID)
	// Clean all rooms connecting to that server
	for roomID, roomServer := range c.roomToWorker {
		if roomServer == workerID {
			wc.Printf("Remove room %s", roomID)
			delete(c.roomToWorker, roomID)
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
