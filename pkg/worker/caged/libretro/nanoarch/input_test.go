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

	for i := 0; i < events; i++ {
		player := rand.Intn(maxPort)
		go func() { state.Input(player, []byte{0, 1}); wg.Done() }()
		go func() { state.IsKeyPressed(uint(player), 100); wg.Done() }()
	}
	wg.Wait()
}

func TestMousePos(t *testing.T) {
	data := []byte{0, 0, 0, 0}

	dx := 1111
	dy := 2222

	binary.BigEndian.PutUint16(data, uint16(dx))
	binary.BigEndian.PutUint16(data[2:], uint16(dy))

	ms := MouseState{}
	ms.ShiftPos(data)

	x := int(ms.PopX())
	y := int(ms.PopY())

	if x != dx || y != dy {
		t.Errorf("invalid state, %v = %v, %v = %v", dx, x, dy, y)
	}

	if ms.dx.Load() != 0 || ms.dy.Load() != 0 {
		t.Errorf("coordinates weren't cleared")
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
