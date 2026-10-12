package api

type (
	CloseRoomRequest        string
	ConnectionRequest[T Id] struct {
		Id      T      `json:"id,omitempty"`
		Addr    string `json:"addr,omitempty"`
		PingURL string `json:"ping_url,omitempty"`
		Port    string `json:"port,omitempty"`
		Tag     string `json:"tag,omitempty"`
		Zone    string `json:"zone,omitempty"`
		IsHTTPS bool   `json:"is_https,omitempty"`
	}
	GetWorkerListResponse struct {
		Servers []Server `json:"servers"`
	}
	RegisterRoomRequest string
)

const (
	DataQueryParam   = "data"
	RoomIdQueryParam = "room_id"
	ZoneQueryParam   = "zone"
	WorkerIdParam    = "wid"
	MatchIdParam     = "match"
	ReplayIdParam    = "replay"
)

// Server contains a list of server groups.
// Server is a separate machine that may contain
// multiple sub-processes.
type Server struct {
	Id       Id     `json:"id,omitempty"`
	Addr     string `json:"addr,omitempty"`
	Machine  string `json:"machine,omitempty"`
	PingURL  string `json:"ping_url"`
	Port     string `json:"port,omitempty"`
	Room     string `json:"room,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Replicas uint32 `json:"replicas,omitempty"`
	IsBusy   bool   `json:"is_busy,omitempty"`
	InGroup  bool   `json:"in_group,omitempty"`
}
