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
	"strings"
)

type (
	Id interface {
		String() string
	}
	Stateful struct {
		Id string `json:"id"`
	}
	Room struct {
		Rid string `json:"room_id"`
	}
	StatefulRoom struct {
		Id  string `json:"id"`
		Rid string `json:"room_id"`
	}
	PT uint8
)

type In[I Id] struct {
	Id      I               `json:"id,omitempty"`
	Payload json.RawMessage `json:"p,omitempty"` // should be json.RawMessage for 2-pass unmarshal
	T       PT              `json:"t"`
}

func (i In[I]) GetId() I           { return i.Id }
func (i In[I]) GetPayload() []byte { return i.Payload }
func (i In[I]) GetType() PT        { return i.T }

type Out struct {
	Payload any    `json:"p,omitempty"`
	Id      string `json:"id,omitempty"` // string because omitempty won't work as intended with arrays
	T       uint8  `json:"t"`
}

func (o *Out) SetId(s string)          { o.Id = s }
func (o *Out) SetType(u uint8)         { o.T = u }
func (o *Out) SetPayload(a any)        { o.Payload = a }
func (o *Out) SetGetId(s fmt.Stringer) { o.Id = s.String() }
func (o *Out) GetPayload() any         { return o.Payload }

// Packet codes:
//
// x, 1xx - user codes
// 15x - webrtc data exchange codes
// 2xx - worker codes
const (
	CheckLatency     PT = 3
	InitSession      PT = 4
	InitWebrtcStream PT = 100
	WebrtcSignal     PT = 101
	StartGame        PT = 104
	QuitGame         PT = 105
	SaveGame         PT = 106
	LoadGame         PT = 107
	ChangePlayer     PT = 108
	RecordGame       PT = 110
	GetWorkerList    PT = 111
	ErrNoFreeSlots   PT = 112
	ResetGame        PT = 113
	LobbyState       PT = 120
	QueueJoin        PT = 121
	QueueLeave       PT = 122
	QueueState       PT = 123
	RoomState        PT = 125
	ChatSend         PT = 126
	ChatMessage      PT = 127
	RoomKick         PT = 128
	RoomAssign       PT = 129
	Kicked           PT = 130
	Notice           PT = 131
	RegisterRoom     PT = 201
	CloseRoom        PT = 202
	TerminateSession PT = 204
	AppVideoChange   PT = 150
	LibNewGameList   PT = 205
	PrevSessions     PT = 206
	MatchResult      PT = 207
)

func (p PT) String() string {
	switch p {
	case CheckLatency:
		return "CheckLatency"
	case InitSession:
		return "InitSession"
	case InitWebrtcStream:
		return "InitWebrtcStream"
	case WebrtcSignal:
		return "WebrtcSignal"
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
	case RecordGame:
		return "RecordGame"
	case GetWorkerList:
		return "GetWorkerList"
	case ErrNoFreeSlots:
		return "NoFreeSlots"
	case ResetGame:
		return "ResetGame"
	case LobbyState:
		return "LobbyState"
	case QueueJoin:
		return "QueueJoin"
	case QueueLeave:
		return "QueueLeave"
	case QueueState:
		return "QueueState"
	case RoomState:
		return "RoomState"
	case ChatSend:
		return "ChatSend"
	case ChatMessage:
		return "ChatMessage"
	case RoomKick:
		return "RoomKick"
	case RoomAssign:
		return "RoomAssign"
	case Kicked:
		return "Kicked"
	case Notice:
		return "Notice"
	case RegisterRoom:
		return "RegisterRoom"
	case CloseRoom:
		return "CloseRoom"
	case TerminateSession:
		return "TerminateSession"
	case AppVideoChange:
		return "AppVideoChange"
	case LibNewGameList:
		return "LibNewGameList"
	case PrevSessions:
		return "PrevSessions"
	case MatchResult:
		return "MatchResult"
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

func Do[I Id, T any](in In[I], fn func(T)) error {
	if dat := Unwrap[T](in.Payload); dat != nil {
		fn(*dat)
		return nil
	}
	return ErrMalformed
}

func DoE[I Id, T any](in In[I], fn func(T) error) error {
	if dat := Unwrap[T](in.Payload); dat != nil {
		return fn(*dat)
	}
	return ErrMalformed
}

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

func Wrap(t any) ([]byte, error) { return json.Marshal(t) }

const separator = "___"

func ExplodeDeepLink(link string) (string, string) {
	p := strings.SplitN(link, separator, 2)

	if len(p) == 1 {
		return p[0], ""
	}

	return p[0], p[1]
}
