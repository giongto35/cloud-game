package tourney

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

// Replay file format (.brpl), little endian:
//
//	"BRPL" | version u8 | header length u32 | header JSON
//	state length u32 | DEFLATE(savestate at frame zero)
//	events length u32 | DEFLATE(events)
//	event = uvarint(frame delta) · u8(port) · u16(button mask)   (only input changes)
//
// Playback restores the frame zero state and applies the logged masks on the
// same frames; the emulation is deterministic, so the match is reproduced exactly.

const (
	replayMagic   = "BRPL"
	replayVersion = 1
	// MaxPorts recorded per frame.
	MaxPorts = 4
)

var ErrBadReplay = errors.New("replay: malformed file")

// Header describes the recorded match.
type Header struct {
	Game      string  `json:"game"`
	System    string  `json:"system"`
	Rule      string  `json:"rule,omitempty"`
	Match     int64   `json:"match,omitempty"`
	Fps       float64 `json:"fps"`
	Frames    uint64  `json:"frames"`
	Winner    int     `json:"winner"`
	CreatedAt int64   `json:"created_at"`
}

// Event is one input change: player Port pressed Mask from Frame on.
type Event struct {
	Frame uint64
	Port  uint8
	Mask  uint16
}

type Replay struct {
	Header Header
	State  []byte
	Events []Event
}

// Recorder logs input changes. Call Observe from the frame hook only.
type Recorder struct {
	last   [MaxPorts]uint16
	events []Event
	frames uint64
}

func NewRecorder() *Recorder { return &Recorder{events: make([]Event, 0, 4096)} }

// Observe records the buttons applied in a frame (relative to the recording start).
func (r *Recorder) Observe(frame uint64, input [MaxPorts]uint16) {
	for p, m := range input {
		if m != r.last[p] {
			r.events = append(r.events, Event{Frame: frame, Port: uint8(p), Mask: m})
			r.last[p] = m
		}
	}
	r.frames = frame + 1
}

func (r *Recorder) Frames() uint64 { return r.frames }

// Events returns a copy of the recorded events.
func (r *Recorder) Events() []Event { return append([]Event(nil), r.events...) }

// Player feeds recorded input back frame by frame.
type Player struct {
	events []Event
	next   int
	cur    [MaxPorts]uint16
	end    uint64
}

func NewPlayer(rp *Replay) *Player {
	return &Player{events: rp.Events, end: rp.Header.Frames}
}

// Input returns the buttons of a frame; ok=false after the last recorded frame.
// Frames must be requested in increasing order.
func (p *Player) Input(frame uint64) (keys [MaxPorts]uint16, ok bool) {
	for p.next < len(p.events) && p.events[p.next].Frame <= frame {
		e := p.events[p.next]
		if int(e.Port) < MaxPorts {
			p.cur[e.Port] = e.Mask
		}
		p.next++
	}
	return p.cur, frame < p.end
}

// Encode writes the replay in the binary format.
func (rp *Replay) Encode(w io.Writer) error {
	hdr, err := json.Marshal(rp.Header)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(w)
	_, _ = bw.WriteString(replayMagic)
	_ = bw.WriteByte(replayVersion)
	writeBlock(bw, hdr)

	state, err := deflate(rp.State)
	if err != nil {
		return err
	}
	writeBlock(bw, state)

	var ev bytes.Buffer
	var prev uint64
	var tmp [binary.MaxVarintLen64]byte
	events := rp.Events
	if !sort.SliceIsSorted(events, func(i, j int) bool { return events[i].Frame < events[j].Frame }) {
		return fmt.Errorf("replay: events must be sorted by frame")
	}
	for _, e := range events {
		n := binary.PutUvarint(tmp[:], e.Frame-prev)
		ev.Write(tmp[:n])
		ev.WriteByte(e.Port)
		var m [2]byte
		binary.LittleEndian.PutUint16(m[:], e.Mask)
		ev.Write(m[:])
		prev = e.Frame
	}
	evz, err := deflate(ev.Bytes())
	if err != nil {
		return err
	}
	writeBlock(bw, evz)
	return bw.Flush()
}

// Bytes encodes the replay into memory.
func (rp *Replay) Bytes() ([]byte, error) {
	var b bytes.Buffer
	err := rp.Encode(&b)
	return b.Bytes(), err
}

// Decode reads a replay written by Encode.
func Decode(data []byte) (*Replay, error) {
	r := bytes.NewReader(data)
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil || string(magic) != replayMagic {
		return nil, ErrBadReplay
	}
	if v, err := r.ReadByte(); err != nil || v != replayVersion {
		return nil, fmt.Errorf("%w: version", ErrBadReplay)
	}
	hdr, err := readBlock(r)
	if err != nil {
		return nil, err
	}
	rp := &Replay{}
	if err := json.Unmarshal(hdr, &rp.Header); err != nil {
		return nil, fmt.Errorf("%w: header", ErrBadReplay)
	}
	statez, err := readBlock(r)
	if err != nil {
		return nil, err
	}
	if rp.State, err = inflate(statez); err != nil {
		return nil, err
	}
	evz, err := readBlock(r)
	if err != nil {
		return nil, err
	}
	ev, err := inflate(evz)
	if err != nil {
		return nil, err
	}
	er := bytes.NewReader(ev)
	var frame uint64
	for er.Len() > 0 {
		d, err := binary.ReadUvarint(er)
		if err != nil {
			return nil, fmt.Errorf("%w: events", ErrBadReplay)
		}
		port, err := er.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("%w: events", ErrBadReplay)
		}
		var m [2]byte
		if _, err := io.ReadFull(er, m[:]); err != nil {
			return nil, fmt.Errorf("%w: events", ErrBadReplay)
		}
		frame += d
		rp.Events = append(rp.Events, Event{Frame: frame, Port: port, Mask: binary.LittleEndian.Uint16(m[:])})
	}
	return rp, nil
}

const maxBlock = 64 << 20

func writeBlock(w *bufio.Writer, b []byte) {
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(b)))
	_, _ = w.Write(n[:])
	_, _ = w.Write(b)
}

func readBlock(r *bytes.Reader) ([]byte, error) {
	var n [4]byte
	if _, err := io.ReadFull(r, n[:]); err != nil {
		return nil, ErrBadReplay
	}
	size := binary.LittleEndian.Uint32(n[:])
	if size > maxBlock || int(size) > r.Len() {
		return nil, ErrBadReplay
	}
	b := make([]byte, size)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, ErrBadReplay
	}
	return b, nil
}

func deflate(b []byte) ([]byte, error) {
	var out bytes.Buffer
	zw, err := flate.NewWriter(&out, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func inflate(b []byte) ([]byte, error) {
	out, err := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(b)), maxBlock))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadReplay, err)
	}
	return out, nil
}
