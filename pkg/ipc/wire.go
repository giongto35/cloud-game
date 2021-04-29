package ipc

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type InPacket struct {
	Id      network.Uid     `json:"id,omitempty"`
	T       PacketType      `json:"t"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type OutPacket struct {
	Id      network.Uid `json:"id,omitempty"`
	T       PacketType  `json:"t"`
	Payload interface{} `json:"payload,omitempty"`
}

type PacketType = uint8
