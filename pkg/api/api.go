package api

import (
	"fmt"

	"github.com/rs/xid"
)

type (
	Stateful struct {
		Id Uid `json:"id"`
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

func StateRoom(id Uid, rid string) StatefulRoom {
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

type Uid struct {
	xid.ID
}

var NilUid = Uid{xid.NilID()}

func NewUid() Uid { return Uid{xid.New()} }

func (u Uid) IsEmpty() bool { return u.IsNil() }
func (u Uid) Short() string { return u.String()[:3] + "." + u.String()[len(u.String())-3:] }

// todo remove null values
