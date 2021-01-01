package api

import "github.com/giongto35/cloud-game/v2/pkg/cws"

const (
	ConfigRequest = "config_request"
	GetRoom       = "get_room"
	CloseRoom     = "close_room"
)

type GameStartRequest struct {
	GameName string `json:"game_name"`
	IsMobile bool   `json:"is_mobile"`
}

func (packet *GameStartRequest) From(data string) error { return from(packet, data) }

type GameStartCall struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

func (packet *GameStartCall) From(data string) error { return from(packet, data) }
func (packet *GameStartCall) To() (string, error)    { return to(packet) }

//
// *** packets ***
//
func ConfigPacket() cws.WSPacket               { return cws.WSPacket{ID: ConfigRequest} }
func GetRoomPacket(data string) cws.WSPacket   { return cws.WSPacket{ID: GetRoom, Data: data} }
func CloseRoomPacket(data string) cws.WSPacket { return cws.WSPacket{ID: CloseRoom, Data: data} }
