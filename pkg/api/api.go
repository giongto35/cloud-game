package api

import (
	"encoding/base64"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/goccy/go-json"
)

type (
	Stateful struct {
		Id network.Uid `json:"id"`
	}
	Room struct {
		Rid string `json:"room_id"` // room id
	}
	StatefulRoom struct {
		Stateful
		Room
	}
	PT uint8
)

type (
	RoomInterface interface {
		GetRoom() string
	}
)

func StateRoom(id network.Uid, rid string) StatefulRoom {
	return StatefulRoom{Stateful: Stateful{id}, Room: Room{rid}}
}
func (sr StatefulRoom) GetRoom() string { return sr.Rid }

// Packet codes:
//
//	x, 1xx - user codes
//	2xx - worker codes
const (
	CheckLatency     PT = 3
	InitSession      PT = 4
	WebrtcInit       PT = 100
	WebrtcOffer      PT = 101
	WebrtcAnswer     PT = 102
	WebrtcIce        PT = 103
	StartGame        PT = 104
	ChangePlayer     PT = 108
	QuitGame         PT = 105
	SaveGame         PT = 106
	LoadGame         PT = 107
	ToggleMultitap   PT = 109
	RecordGame       PT = 110
	GetWorkerList    PT = 111
	RegisterRoom     PT = 201
	CloseRoom        PT = 202
	IceCandidate        = WebrtcIce
	TerminateSession PT = 204
)

func (p PT) String() string {
	switch p {
	case CheckLatency:
		return "CheckLatency"
	case InitSession:
		return "InitSession"
	case WebrtcInit:
		return "WebrtcInit"
	case WebrtcOffer:
		return "WebrtcOffer"
	case WebrtcAnswer:
		return "WebrtcAnswer"
	case WebrtcIce:
		return "WebrtcIce"
	case StartGame:
		return "StartGame"
	case ChangePlayer:
		return "ChangePlayer"
	case QuitGame:
		return "QuitGame"
	case SaveGame:
		return "SaveGame"
	case LoadGame:
		return "LoadGame"
	case ToggleMultitap:
		return "ToggleMultitap"
	case RecordGame:
		return "RecordGame"
	case GetWorkerList:
		return "GetWorkerList"
	case RegisterRoom:
		return "RegisterRoom"
	case CloseRoom:
		return "CloseRoom"
	case TerminateSession:
		return "TerminateSession"
	default:
		return "Unknown"
	}
}

// Various codes
const (
	EMPTY = ""
	OK    = "ok"
)

var (
	ErrForbidden = fmt.Errorf("forbidden")
	ErrMalformed = fmt.Errorf("malformed")
)

func Unwrap[T any](data []byte) *T {
	out := new(T)
	if err := json.Unmarshal(data, out); err != nil {
		return nil
	}
	return out
}

func UnwrapChecked[T any](bytes []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	return Unwrap[T](bytes), nil
}

// ToBase64Json encodes data to a URL-encoded Base64+JSON string.
func ToBase64Json(data any) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// FromBase64Json decodes data from a URL-encoded Base64+JSON string.
func FromBase64Json(data string, obj any) error {
	b, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
}
