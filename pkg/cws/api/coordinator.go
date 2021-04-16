package api

import (
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

const (
	ConfigRequest = "config_request"
	GetRoom       = "get_room"
	CloseRoom     = "close_room"
	RegisterRoom  = "register_room"
	Heartbeat     = "heartbeat"
	IceCandidate  = "ice_candidate"

	NoData = ""

	InitWebrtc = "init_webrtc"
	Answer     = "answer"
	Offer      = "offer"

	GameStart        = "start"
	GameQuit         = "quit"
	GameSave         = "save"
	GameLoad         = "load"
	GamePlayerSelect = "player_index"
	GameMultitap     = "multitap"
)

type GameStartRequest struct {
	GameName string `json:"game_name"`
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
func ConfigPacket() cws.WSPacket                  { return cws.WSPacket{ID: ConfigRequest} }
func RegisterRoomPacket(data string) cws.WSPacket { return cws.WSPacket{ID: RegisterRoom, Data: data} }
func GetRoomPacket(data string) cws.WSPacket      { return cws.WSPacket{ID: GetRoom, Data: data} }
func CloseRoomPacket(data string) cws.WSPacket    { return cws.WSPacket{ID: CloseRoom, Data: data} }
func IceCandidatePacket(data string, sessionId network.Uid) cws.WSPacket {
	return cws.WSPacket{ID: IceCandidate, Data: data, SessionID: sessionId}
}
func OfferPacket(sdp string) cws.WSPacket { return cws.WSPacket{ID: "offer", Data: sdp} }
