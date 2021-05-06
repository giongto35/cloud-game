package api

type ConnectionRequest struct {
	Zone     string `json:"zone,omitempty"`
	PingAddr string `json:"ping_addr,omitempty"`
}
type RegisterRoomRequest = string
type CloseRoomRequest = string
