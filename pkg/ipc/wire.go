package ipc

import "github.com/giongto35/cloud-game/v2/pkg/network"

type Packet struct {
	Id      network.Uid `json:"id,omitempty"`
	T       PacketType  `json:"t"`
	Payload interface{} `json:"payload,omitempty"`
}

type PacketType uint8

type Call struct {
	done     chan struct{}
	err      error
	Request  Packet
	Response Packet
}
