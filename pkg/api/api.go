package api

import (
	"errors"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/goccy/go-json"
)

type (
	PType    = uint8
	Stateful struct {
		Id network.Uid `json:"id"`
	}
)

// Various codes

const (
	EMPTY = ""
	OK    = "ok"
)

// User
const (
	CheckLatency       PType = 3
	InitSession        PType = 4
	WebrtcInit         PType = 100
	WebrtcOffer        PType = 101
	WebrtcAnswer       PType = 102
	WebrtcIceCandidate PType = 103
	StartGame          PType = 104
	ChangePlayer       PType = 108
	QuitGame           PType = 105
	SaveGame           PType = 106
	LoadGame           PType = 107
	ToggleMultitap     PType = 109
	RecordGame         PType = 110
	GetWorkerList      PType = 111
)

// Worker
const (
	RegisterRoom     PType = 2
	CloseRoom        PType = 3
	IceCandidate           = WebrtcIceCandidate
	TerminateSession PType = 5
)

var ErrMalformed = errors.New("malformed")

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
