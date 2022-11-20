package api

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/network"
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
	ERROR = "error"
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

func Unwrap[T any](bytes []byte) (*T, error) {
	out := new(T)
	if err := json.Unmarshal(bytes, out); err != nil {
		return nil, err
	}
	return out, nil
}

func UnwrapChecked[T any](bytes []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	return Unwrap[T](bytes)
}
