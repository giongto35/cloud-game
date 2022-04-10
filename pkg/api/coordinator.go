package api

type (
	CloseRoomRequest  = string
	ConnectionRequest struct {
		Addr    string `json:"addr,omitempty"`
		Id      string `json:"id,omitempty"`
		IsHTTPS bool   `json:"is_https,omitempty"`
		PingURL string `json:"ping_url,omitempty"`
		Port    string `json:"port,omitempty"`
		Tag     string `json:"tag,omitempty"`
		Zone    string `json:"zone,omitempty"`
	}
	GetWorkerListRequest  struct{}
	GetWorkerListResponse struct {
		Servers []Server `json:"servers"`
	}
	RegisterRoomRequest = string
)

// Server contains a list of server groups.
// Server is a separate machine that may contain
// multiple sub-processes.
type Server struct {
	Addr     string `json:"addr,omitempty"`
	Id       string `json:"id,omitempty"`
	IsBusy   bool   `json:"is_busy,omitempty"`
	InGroup  bool   `json:"in_group,omitempty"`
	PingURL  string `json:"ping_url"`
	Port     string `json:"port,omitempty"`
	Replicas uint32 `json:"replicas,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Zone     string `json:"zone,omitempty"`
}

const (
	DataQueryParam   = "data"
	RoomIdQueryParam = "room_id"
	ZoneQueryParam   = "zone"
	WorkerIdParam    = "wid"
)
