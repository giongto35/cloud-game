package rtp

import (
	"math/rand"
	"time"
)

// Payloader payloads a byte array for use as rtp.Packet payloads
type Payloader interface {
	Payload(mtu int, payload []byte) [][]byte
}

// Packetizer packetizes a payload
type Packetizer interface {
	Packetize(payload []byte, samples uint32) []*Packet
	EnableAbsSendTime(value int)
}

type packetizer struct {
	MTU              int
	PayloadType      uint8
	SSRC             uint32
	Payloader        Payloader
	Sequencer        Sequencer
	Timestamp        uint32
	ClockRate        uint32
	extensionNumbers struct { //put extension numbers in here. If they're 0, the extension is disabled (0 is not a legal extension number)
		AbsSendTime int //http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
	}
	timegen func() time.Time
}

// NewPacketizer returns a new instance of a Packetizer for a specific payloader
func NewPacketizer(mtu int, pt uint8, ssrc uint32, payloader Payloader, sequencer Sequencer, clockRate uint32) Packetizer {
	rs := rand.NewSource(time.Now().UnixNano())
	r := rand.New(rs)

	return &packetizer{
		MTU:         mtu,
		PayloadType: pt,
		SSRC:        ssrc,
		Payloader:   payloader,
		Sequencer:   sequencer,
		Timestamp:   r.Uint32(),
		ClockRate:   clockRate,
		timegen:     time.Now,
	}
}

func (p *packetizer) EnableAbsSendTime(value int) {
	p.extensionNumbers.AbsSendTime = value
}

func toNtpTime(t time.Time) uint64 {
	var s uint64
	var f uint64
	u := uint64(t.UnixNano())
	s = u / 1e9
	s += 0x83AA7E80 //offset in seconds between unix epoch and ntp epoch
	f = u % 1e9
	f <<= 32
	f /= 1e9
	s <<= 32

	return s | f
}

// Packetize packetizes the payload of an RTP packet and returns one or more RTP packets
func (p *packetizer) Packetize(payload []byte, samples uint32) []*Packet {
	// Guard against an empty payload
	if len(payload) == 0 {
		return nil
	}

	payloads := p.Payloader.Payload(p.MTU-12, payload)
	packets := make([]*Packet, len(payloads))

	for i, pp := range payloads {
		packets[i] = &Packet{
			Header: Header{
				Version:        2,
				Padding:        false,
				Extension:      false,
				Marker:         i == len(payloads)-1,
				PayloadType:    p.PayloadType,
				SequenceNumber: p.Sequencer.NextSequenceNumber(),
				Timestamp:      p.Timestamp, // Figure out how to do timestamps
				SSRC:           p.SSRC,
			},
			Payload: pp,
		}
	}
	p.Timestamp += samples

	if len(packets) != 0 && p.extensionNumbers.AbsSendTime != 0 {
		t := toNtpTime(p.timegen()) >> 14
		//apply http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
		packets[len(packets)-1].Header.Extension = true
		packets[len(packets)-1].ExtensionProfile = 0xBEDE
		packets[len(packets)-1].ExtensionPayload = []byte{
			//the first byte is
			// 0 1 2 3 4 5 6 7
			//+-+-+-+-+-+-+-+-+
			//|  ID   |  len  |
			//+-+-+-+-+-+-+-+-+
			//per RFC 5285
			//Len is the number of bytes in the extension - 1

			byte((p.extensionNumbers.AbsSendTime << 4) | 2),
			byte(t & 0xFF0000 >> 16),
			byte(t & 0xFF00 >> 8),
			byte(t & 0xFF),
		}

	}

	return packets
}
