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

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/cws"
	"github.com/giongto35/cloud-game/overlord/gamelist"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

const (
	gameboyIndex = "./static/game.html"
	gamePath     = "games"
)

type Server struct {
	roomToServer map[string]string
	// workerClients are the map serverID to worker Client
	workerClients map[string]*WorkerClient
}

var upgrader = websocket.Upgrader{}
var errNotFound = errors.New("Not found")

func NewServer() *Server {
	return &Server{
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

	//bs, err := ioutil.ReadFile(indexFN)
	//if err != nil {
	//log.Fatal(err)
	//}
	//w.Write(bs)
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
	client := NewWorkerClient(c, serverID, getRemoteAddress(c))
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

	// Set up server
	// SessionID will be the unique per frontend connection
	sessionID := uuid.Must(uuid.NewV4()).String()
	var serverID string
	if config.MatchWorkerRandom {
		serverID, err = o.findBestServerRandom()
	} else {
		//serverID, err = o.findBestServer(frontendAddr)
		serverID, err = o.findBestServerFromBrowser(client)
	}

	if err != nil {
		log.Println(err)
		return
	}

	// Setup session
	wssession := &Session{
		ID:            sessionID,
		handler:       o,
		BrowserClient: client,
		WorkerClient:  o.workerClients[serverID],
		ServerID:      serverID,
	}
	// TODO:?
	//defer wssession.Close()
	log.Println("New client will conect to server", wssession.ServerID)

	wssession.RouteBrowser()

	wssession.BrowserClient.Send(cws.WSPacket{
		ID:   "gamelist",
		Data: gamelist.GetEncodedGameList(gamePath),
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
}

// findBestServer returns the best server for a session
func (o *Server) findBestServerRandom() (string, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(o.workerClients) == 0 {
		return "", errors.New("No server found")
	}

	r := rand.Intn(len(o.workerClients))
	for k, _ := range o.workerClients {
		if r == 0 {
			return k, nil
		}
		r--
	}

	return "", errors.New("No server found")
}

// findBestServerFromBrowser returns the best server for a session
// All workers addresses are sent to user and user will ping
func (o *Server) findBestServerFromBrowser(client *BrowserClient) (string, error) {
	// TODO: Find best Server by latency, currently return by ping
	if len(o.workerClients) == 0 {
		return "", errors.New("No server found")
	}

	// TODO: Add timeout
	log.Println("Ping worker to get latency for ", client)
	latencies := o.getLatencyMapFromBrowser(client)
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
func (o *Server) getLatencyMapFromBrowser(client *BrowserClient) map[*WorkerClient]int64 {
	workersList := []*WorkerClient{}

	latencyMap := map[*WorkerClient]int64{}

	// addressList is the list of worker addresses
	addressList := []string{}
	for _, workerClient := range o.workerClients {
		workersList = append(workersList, workerClient)
		addressList = append(addressList, workerClient.Address)
	}

	// send this address to user and get back latency
	log.Println("Send sync", addressList, strings.Join(addressList, ","))
	data := client.SyncSend(cws.WSPacket{
		ID:   "checkLatency",
		Data: strings.Join(addressList, ","),
	})

	fmt.Println("???", data)
	respLatency := map[string]interface{}{}
	err := json.Unmarshal([]byte(data.Data), &respLatency)
	if err != nil {
		log.Println(err)
		return latencyMap
	}
	//log.Println("Received latency map:", data.Data)
	//latencies := strings.Split(data.Data, ",")
	//log.Println("Received latency list:", latencies)

	//for _, workerClient := range workersList {
	////il, _ := strconv.Atoi(latencies[i])
	//if latency, ok := respLatency[workerClient.Address]; ok {
	//latencyMap[workerClient] = latency
	//}
	//}
	return latencyMap
}

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

// getRemoteAddress returns public address of websocket connection
func getRemoteAddress(conn *websocket.Conn) string {
	var remoteAddr string
	log.Println(conn.RemoteAddr().String())
	if parts := strings.Split(conn.RemoteAddr().String(), ":"); len(parts) == 2 {
		remoteAddr = parts[0]
	}
	if remoteAddr == "" {
		return "localhost"
	}

	return remoteAddr
}
