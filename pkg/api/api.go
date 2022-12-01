package api

import (
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/goccy/go-json"
)

type Stateful struct {
	Id network.Uid `json:"id"`
}

type PT uint8

// Packet codes:
//
//	x, 1xx - user codes
//	2xx - worker codes
const (
	CheckLatency       PT = 3
	InitSession        PT = 4
	WebrtcInit         PT = 100
	WebrtcOffer        PT = 101
	WebrtcAnswer       PT = 102
	WebrtcIceCandidate PT = 103
	StartGame          PT = 104
	ChangePlayer       PT = 108
	QuitGame           PT = 105
	SaveGame           PT = 106
	LoadGame           PT = 107
	ToggleMultitap     PT = 109
	RecordGame         PT = 110
	GetWorkerList      PT = 111
	RegisterRoom       PT = 201
	CloseRoom          PT = 202
	IceCandidate          = WebrtcIceCandidate
	TerminateSession   PT = 204
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
	case WebrtcIceCandidate:
		return "WebrtcIceCandidate"
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
