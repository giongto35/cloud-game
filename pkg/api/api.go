// Package api defines the general API for both coordinator and worker applications.
//
// Each API call (request and response) is a JSON-encoded "packet" of the following structure:
//
//	id - (optional) a globally unique packet id;
//	 t - (required) one of the predefined unique packet types;
//	 p - (optional) packet payload with arbitrary data.
//
// The basic idea behind this API is that the packets differentiate by their predefined types
// with which it is possible to unwrap the payload into distinct request/response data structures.
// And the id field is used for tracking packets through a chain of different network points (apps, devices),
// for example, passing a packet from a browser forward to a worker and back through a coordinator.
//
// Example:
//
//	{"t":4,"p":{"ice":[{"urls":"stun:stun.l.google.com:19302"}],"games":["Sushi The Cat"],"wid":"cfv68irdrc3ifu3jn6bg"}}
package api

import (
	"encoding/json"
	"fmt"
)

type (
	Id interface {
		String() string
	}
	Stateful[T Id] struct {
		Id T `json:"id"`
	}
	Room struct {
		Rid string `json:"room_id"` // room id
	}
	StatefulRoom[T Id] struct {
		Stateful[T]
		Room
	}
	PT uint8
)

type In[I Id] struct {
	Id      I               `json:"id,omitempty"`
	T       PT              `json:"t"`
	Payload json.RawMessage `json:"p,omitempty"` // should be json.RawMessage for 2-pass unmarshal
}

func (i In[I]) GetId() I           { return i.Id }
func (i In[I]) GetPayload() []byte { return i.Payload }
func (i In[I]) GetType() PT        { return i.T }

type Out struct {
	Id      string `json:"id,omitempty"` // string because omitempty won't work as intended with arrays
	T       uint8  `json:"t"`
	Payload any    `json:"p,omitempty"`
}

func (o *Out) SetId(s string)          { o.Id = s }
func (o *Out) SetType(u uint8)         { o.T = u }
func (o *Out) SetPayload(a any)        { o.Payload = a }
func (o *Out) SetGetId(s fmt.Stringer) { o.Id = s.String() }
func (o *Out) GetPayload() any         { return o.Payload }

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
	ErrNoFreeSlots   PT = 112
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

var (
	EmptyPacket = Out{Payload: ""}
	ErrPacket   = Out{Payload: "err"}
	OkPacket    = Out{Payload: "ok"}
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
