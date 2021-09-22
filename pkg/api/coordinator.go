package api

type ConnectionRequest struct {
	Zone     string `json:"zone,omitempty"`
	PingAddr string `json:"ping_addr,omitempty"`
	IsHTTPS  bool   `json:"is_https,omitempty"`
}
type RegisterRoomRequest = string
type CloseRoomRequest = string
