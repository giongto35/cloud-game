package nanoarch

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"
)

func TestConcurrentInput(t *testing.T) {
	var wg sync.WaitGroup
	state := InputState{}
	events := 1000
	wg.Add(2 * events)

	for range events {
		player := rand.Intn(maxPort)
		go func() { state.Input(player, []byte{0, 1}); wg.Done() }()
		go func() { state.IsKeyPressed(uint(player), 100); wg.Done() }()
	}
	wg.Wait()
}

func TestMousePos(t *testing.T) {
	tests := []struct {
		name string
		dx   int16
		dy   int16
		rx   int16
		ry   int16
		b    func(dx, dy int16) []byte
	}{
		{name: "normal", dx: -10123, dy: 5678, rx: -10123, ry: 5678, b: func(dx, dy int16) []byte {
			data := []byte{0, 0, 0, 0}
			binary.BigEndian.PutUint16(data, uint16(dx))
			binary.BigEndian.PutUint16(data[2:], uint16(dy))
			return data
		}},
		{name: "wrong endian", dx: -1234, dy: 5678, rx: 12027, ry: 11798, b: func(dx, dy int16) []byte {
			data := []byte{0, 0, 0, 0}
			binary.LittleEndian.PutUint16(data, uint16(dx))
			binary.LittleEndian.PutUint16(data[2:], uint16(dy))
			return data
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := test.b(test.dx, test.dy)

			ms := MouseState{}
			ms.ShiftPos(data)

			x := int16(ms.PopX())
			y := int16(ms.PopY())

			if x != test.rx || y != test.ry {
				t.Errorf("invalid state, %v = %v, %v = %v", test.rx, x, test.ry, y)
			}

			if ms.dx.Load() != 0 || ms.dy.Load() != 0 {
				t.Errorf("coordinates weren't cleared")
			}
		})
	}
}

func TestMouseButtons(t *testing.T) {
	tests := []struct {
		name string
		data byte
		l    bool
		r    bool
		m    bool
	}{
		{name: "l+r+m+", data: 1 + 2 + 4, l: true, r: true, m: true},
		{name: "l-r-m-", data: 0},
		{name: "l-r+m-", data: 2, r: true},
		{name: "l+r-m+", data: 1 + 4, l: true, m: true},
	}

	ms := MouseState{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms.SetButtons(test.data)
			l, r, m := ms.Buttons()
			if l != test.l || r != test.r || m != test.m {
				t.Errorf("wrong button state: %v -> %v, %v, %v", test.data, l, r, m)
			}
		})
	}
}
