package api

import "github.com/giongto35/cloud-game/v2/pkg/cws"

const (
	GetRoom      = "get_room"
	CloseRoom    = "close_room"
	RegisterRoom = "register_room"
	Heartbeat    = "heartbeat"
	IceCandidate = "ice_candidate"

	NoData = ""

	InitWebrtc = "init_webrtc"
	Answer     = "answer"

	GameStart        = "start"
	GameQuit         = "quit"
	GameSave         = "save"
	GameLoad         = "load"
	GamePlayerSelect = "player_index"
	GameMultitap     = "multitap"
	GameRecording    = "recording"
	GetServerList    = "get_server_list"
)

type GameStartRequest struct {
	GameName   string `json:"game_name"`
	Record     bool   `json:"record,omitempty"`
	RecordUser string `json:"record_user,omitempty"`
}

func (packet *GameStartRequest) From(data string) error { return from(packet, data) }

type GameRecordingRequest struct {
	Active bool   `json:"active"`
	User   string `json:"user"`
}

func (packet *GameRecordingRequest) From(data string) error { return from(packet, data) }

type GameStartCall struct {
	Name       string `json:"name"`
	Base       string `json:"base"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Record     bool   `json:"record,omitempty"`
	RecordUser string `json:"record_user,omitempty"`
}

func (packet *GameStartCall) From(data string) error { return from(packet, data) }
func (packet *GameStartCall) To() (string, error)    { return to(packet) }

type ConnectionRequest struct {
	Addr    string `json:"addr,omitempty"`
	IsHTTPS bool   `json:"is_https,omitempty"`
	PingURL string `json:"ping_url,omitempty"`
	Port    string `json:"port,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Zone    string `json:"zone,omitempty"`
}

type GetServerListRequest struct{}
type GetServerListResponse struct {
	Servers []Server `json:"servers"`
}

// Server contains a list of server groups.
// Server is a separate machine that may contain
// multiple sub-processes.
type Server struct {
	Addr     string `json:"addr,omitempty"`
	Id       string `json:"id,omitempty"`
	IsBusy   bool   `json:"is_busy,omitempty"`
	PingURL  string `json:"ping_url"`
	Port     string `json:"port,omitempty"`
	Replicas uint32 `json:"replicas,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Zone     string `json:"zone,omitempty"`
}

func (packet *GetServerListRequest) From(data string) error { return from(packet, data) }
func (packet *GetServerListResponse) To() (string, error)   { return to(packet) }

// packets

func RegisterRoomPacket(data string) cws.WSPacket { return cws.WSPacket{ID: RegisterRoom, Data: data} }
func GetRoomPacket(data string) cws.WSPacket      { return cws.WSPacket{ID: GetRoom, Data: data} }
func CloseRoomPacket(data string) cws.WSPacket    { return cws.WSPacket{ID: CloseRoom, Data: data} }
func IceCandidatePacket(data string, sessionId string) cws.WSPacket {
	return cws.WSPacket{ID: IceCandidate, Data: data, SessionID: sessionId}
}
