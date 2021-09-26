package api

import "github.com/giongto35/cloud-game/v2/pkg/network"

// User
const (
	CheckLatency       uint8 = 3
	InitSession        uint8 = 4
	WebrtcInit         uint8 = 100
	WebrtcOffer        uint8 = 101
	WebrtcAnswer       uint8 = 102
	WebrtcIceCandidate uint8 = 103
	StartGame          uint8 = 104
	ChangePlayer       uint8 = 108
	QuitGame           uint8 = 105
	SaveGame           uint8 = 106
	LoadGame           uint8 = 107
	ToggleMultitap     uint8 = 109
)

// Worker
const (
	RegisterRoom     uint8 = 2
	CloseRoom        uint8 = 3
	IceCandidate     uint8 = 4
	TerminateSession uint8 = 5
)

type Stateful struct {
	Id network.Uid `json:"id"`
}
