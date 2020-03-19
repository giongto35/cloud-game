package coordinator

// Session represents a session connected from the browser to the current server
// It requires one connection to browser and one connection to the coordinator
// connection to browser is 1-1. connection to coordinator is n - 1
// Peerconnection can be from other server to ensure better latency
type Session struct {
	ID            string
	BrowserClient *BrowserClient
	WorkerClient  *WorkerClient
	// CoordinatorClient *CoordinatorClient
	// peerconnection *webrtc.WebRTC

	// TODO: Decouple this
	handler *Server

	ServerID string
	RoomID   string
}
