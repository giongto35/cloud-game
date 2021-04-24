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

type PacketOption func(opt *Packet)

func Id(id network.Uid) PacketOption { return func(p *Packet) { p.Id = id } }

func T(t uint8) PacketOption               { return func(p *Packet) { p.T = PacketType(t) } }
func Payload(pld interface{}) PacketOption { return func(p *Packet) { p.Payload = pld } }
