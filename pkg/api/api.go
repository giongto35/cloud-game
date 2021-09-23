package api

import "github.com/giongto35/cloud-game/v2/pkg/network"

// User
const (
	CheckLatency       uint8 = 3   // out
	InitSession        uint8 = 4   // out
	WebrtcInit         uint8 = 100 // in
	WebrtcOffer        uint8 = 101 // out
	WebrtcAnswer       uint8 = 102 // in
	WebrtcIceCandidate uint8 = 103 // in / out
	StartGame          uint8 = 104 // in / out
	ChangePlayer       uint8 = 108 // in / out
	QuitGame           uint8 = 105 // in
	SaveGame           uint8 = 106 // in
	LoadGame           uint8 = 107 // in
	ToggleMultitap     uint8 = 109 // in
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
