package ipc

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type InPacket struct {
	Id      network.Uid     `json:"id,omitempty"`
	T       PacketType      `json:"t"`
	Payload json.RawMessage `json:"p,omitempty"`
}

type OutPacket struct {
	Id      network.Uid `json:"id,omitempty"`
	T       PacketType  `json:"t"`
	Payload interface{} `json:"p,omitempty"`
}

type PacketType = uint8

func (p InPacket) Proxy(payload interface{}) OutPacket {
	return OutPacket{Id: p.Id, T: p.T, Payload: payload}
}
