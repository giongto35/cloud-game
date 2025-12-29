package nanoarch

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"
)

func TestInputState_SetInput(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		data     []byte
		keys     uint32
		axes     [4]int16
		triggers [2]int16
	}{
		{
			name: "buttons only",
			port: 0,
			data: []byte{0xFF, 0x01},
			keys: 0x01FF,
		},
		{
			name: "buttons and axes",
			port: 1,
			data: []byte{0x03, 0x00, 0x10, 0x27, 0xF0, 0xD8, 0x00, 0x80, 0xFF, 0x7F},
			keys: 0x0003,
			axes: [4]int16{10000, -10000, -32768, 32767},
		},
		{
			name: "partial axes",
			port: 2,
			data: []byte{0x01, 0x00, 0x64, 0x00},
			keys: 0x0001,
			axes: [4]int16{100, 0, 0, 0},
		},
		{
			name: "max port",
			port: 3,
			data: []byte{0xFF, 0xFF},
			keys: 0xFFFF,
		},
		{
			name: "full input with triggers",
			port: 0,
			data: []byte{
				0x03, 0x00, // buttons
				0x10, 0x27, // LX: 10000
				0xF0, 0xD8, // LY: -10000
				0x00, 0x80, // RX: -32768
				0xFF, 0x7F, // RY: 32767
				0xFF, 0x3F, // L2: 16383
				0xFF, 0x7F, // R2: 32767
			},
			keys:     0x0003,
			axes:     [4]int16{10000, -10000, -32768, 32767},
			triggers: [2]int16{16383, 32767},
		},
		{
			name: "axes without triggers",
			port: 1,
			data: []byte{
				0x01, 0x00,
				0x64, 0x00, // LX: 100
				0xC8, 0x00, // LY: 200
				0x2C, 0x01, // RX: 300
				0x90, 0x01, // RY: 400
			},
			keys: 0x0001,
			axes: [4]int16{100, 200, 300, 400},
		},
		{
			name: "zero triggers",
			port: 2,
			data: []byte{
				0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, // L2: 0
				0x00, 0x00, // R2: 0
			},
			keys: 0x0000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := InputState{}
			state.SetInput(test.port, test.data)

			if state[test.port].keys != test.keys {
				t.Errorf("keys: got %v, want %v", state[test.port].keys, test.keys)
			}

			// Check axes from packed int64
			axes := state[test.port].axes
			for i, want := range test.axes {
				got := int16(axes >> (i * 16))
				if got != want {
					t.Errorf("axes[%d]: got %v, want %v", i, got, want)
				}
			}

			// Check triggers from packed int32
			triggers := state[test.port].triggers
			l2 := int16(triggers)
			r2 := int16(triggers >> 16)
			if l2 != test.triggers[0] {
				t.Errorf("L2: got %v, want %v", l2, test.triggers[0])
			}
			if r2 != test.triggers[1] {
				t.Errorf("R2: got %v, want %v", r2, test.triggers[1])
			}
		})
	}
}

func TestInputState_AxisExtraction(t *testing.T) {
	state := InputState{}
	data := []byte{
		0x00, 0x00, // buttons
		0x01, 0x00, // LX: 1
		0x02, 0x00, // LY: 2
		0x03, 0x00, // RX: 3
		0x04, 0x00, // RY: 4
		0x05, 0x00, // L2: 5
		0x06, 0x00, // R2: 6
	}
	state.SetInput(0, data)

	axes := state[0].axes
	expected := []int16{1, 2, 3, 4}
	for i, want := range expected {
		got := int16(axes >> (i * 16))
		if got != want {
			t.Errorf("axis[%d]: got %v, want %v", i, got, want)
		}
	}

	triggers := state[0].triggers
	if got := int16(triggers); got != 5 {
		t.Errorf("L2: got %v, want 5", got)
	}
	if got := int16(triggers >> 16); got != 6 {
		t.Errorf("R2: got %v, want 6", got)
	}
}

func TestInputState_NegativeAxes(t *testing.T) {
	state := InputState{}
	data := []byte{
		0x00, 0x00, // buttons
		0x00, 0x80, // LX: -32768
		0xFF, 0xFF, // LY: -1
		0x01, 0x80, // RX: -32767
		0xFE, 0xFF, // RY: -2
	}
	state.SetInput(0, data)

	axes := state[0].axes
	expected := []int16{-32768, -1, -32767, -2}
	for i, want := range expected {
		got := int16(axes >> (i * 16))
		if got != want {
			t.Errorf("axis[%d]: got %v, want %v", i, got, want)
		}
	}
}

func TestInputState_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	state := InputState{}
	events := 1000
	wg.Add(events)

	for range events {
		player := rand.Intn(maxPort)
		go func() {
			// Full 14-byte input
			state.SetInput(player, []byte{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestKeyboardState_SetKey(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		pressed bool
		key     uint
		mod     uint16
	}{
		{
			name:    "key pressed",
			data:    []byte{0, 0, 0, 42, 1, 0, 3},
			pressed: true,
			key:     42,
			mod:     3,
		},
		{
			name:    "key released",
			data:    []byte{0, 0, 0, 100, 0, 0, 0},
			pressed: false,
			key:     100,
			mod:     0,
		},
		{
			name:    "high key code",
			data:    []byte{0, 0, 1, 50, 1, 0xFF, 0xFF},
			pressed: true,
			key:     306,
			mod:     0xFFFF,
		},
		{
			name:    "invalid length",
			data:    []byte{0, 0, 0},
			pressed: false,
			key:     0,
			mod:     0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ks := KeyboardState{}
			pressed, key, mod := ks.SetKey(test.data)

			if pressed != test.pressed {
				t.Errorf("pressed: got %v, want %v", pressed, test.pressed)
			}
			if key != test.key {
				t.Errorf("key: got %v, want %v", key, test.key)
			}
			if mod != test.mod {
				t.Errorf("mod: got %v, want %v", mod, test.mod)
			}
		})
	}
}

func TestKeyboardState_IsPressed(t *testing.T) {
	ks := KeyboardState{}

	// Initially not pressed
	if ks.keys[0].Load() != 0 {
		t.Error("key should not be pressed initially")
	}

	// Press key
	ks.SetKey([]byte{0, 0, 0, 42, 1, 0, 0})
	if (ks.keys[42/64].Load()>>(42%64))&1 != 1 {
		t.Error("key should be pressed")
	}

	// Release key
	ks.SetKey([]byte{0, 0, 0, 42, 0, 0, 0})
	if (ks.keys[42/64].Load()>>(42%64))&1 != 0 {
		t.Error("key should be released")
	}
}

func TestKeyboardState_MultipleBits(t *testing.T) {
	ks := KeyboardState{}

	// Press keys in different uint64 slots
	keys := []uint{0, 63, 64, 127, 128, 200, 300, 341}
	for _, k := range keys {
		data := make([]byte, 7)
		binary.BigEndian.PutUint32(data, uint32(k))
		data[4] = 1
		ks.SetKey(data)
	}

	// Check all pressed
	for _, k := range keys {
		if (ks.keys[k/64].Load()>>(k%64))&1 != 1 {
			t.Errorf("key %d should be pressed", k)
		}
	}

	// Release some
	for _, k := range []uint{0, 128, 341} {
		data := make([]byte, 7)
		binary.BigEndian.PutUint32(data, uint32(k))
		data[4] = 0
		ks.SetKey(data)
	}

	// Check states
	expected := map[uint]uint64{
		0: 0, 63: 1, 64: 1, 127: 1, 128: 0, 200: 1, 300: 1, 341: 0,
	}
	for k, want := range expected {
		got := (ks.keys[k/64].Load() >> (k % 64)) & 1
		if got != want {
			t.Errorf("key %d: got %v, want %v", k, got, want)
		}
	}
}

func TestKeyboardState_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	ks := KeyboardState{}
	events := 1000
	wg.Add(events * 2)

	for range events {
		key := uint(rand.Intn(RetrokLast))
		go func() {
			data := make([]byte, 7)
			binary.BigEndian.PutUint32(data, uint32(key))
			data[4] = byte(rand.Intn(2))
			ks.SetKey(data)
			wg.Done()
		}()
		go func() {
			_ = (ks.keys[key/64].Load() >> (key % 64)) & 1
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestMouseState_ShiftPos(t *testing.T) {
	tests := []struct {
		name string
		dx   int16
		dy   int16
		rx   int16
		ry   int16
		b    func(dx, dy int16) []byte
	}{
		{
			name: "positive values",
			dx:   100,
			dy:   200,
			rx:   100,
			ry:   200,
			b: func(dx, dy int16) []byte {
				data := make([]byte, 4)
				binary.BigEndian.PutUint16(data, uint16(dx))
				binary.BigEndian.PutUint16(data[2:], uint16(dy))
				return data
			},
		},
		{
			name: "negative values",
			dx:   -10123,
			dy:   5678,
			rx:   -10123,
			ry:   5678,
			b: func(dx, dy int16) []byte {
				data := make([]byte, 4)
				binary.BigEndian.PutUint16(data, uint16(dx))
				binary.BigEndian.PutUint16(data[2:], uint16(dy))
				return data
			},
		},
		{
			name: "wrong endian",
			dx:   -1234,
			dy:   5678,
			rx:   12027,
			ry:   11798,
			b: func(dx, dy int16) []byte {
				data := make([]byte, 4)
				binary.LittleEndian.PutUint16(data, uint16(dx))
				binary.LittleEndian.PutUint16(data[2:], uint16(dy))
				return data
			},
		},
		{
			name: "max values",
			dx:   32767,
			dy:   -32768,
			rx:   32767,
			ry:   -32768,
			b: func(dx, dy int16) []byte {
				data := make([]byte, 4)
				binary.BigEndian.PutUint16(data, uint16(dx))
				binary.BigEndian.PutUint16(data[2:], uint16(dy))
				return data
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := MouseState{}
			ms.ShiftPos(test.b(test.dx, test.dy))

			x, y := int16(ms.dx.Swap(0)), int16(ms.dy.Swap(0))

			if x != test.rx || y != test.ry {
				t.Errorf("got (%v, %v), want (%v, %v)", x, y, test.rx, test.ry)
			}

			if ms.dx.Load() != 0 || ms.dy.Load() != 0 {
				t.Error("coordinates weren't cleared")
			}
		})
	}
}

func TestMouseState_ShiftPosAccumulates(t *testing.T) {
	ms := MouseState{}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data, uint16(10))
	binary.BigEndian.PutUint16(data[2:], uint16(20))

	ms.ShiftPos(data)
	ms.ShiftPos(data)
	ms.ShiftPos(data)

	if got := ms.dx.Load(); got != 30 {
		t.Errorf("dx: got %v, want 30", got)
	}
	if got := ms.dy.Load(); got != 60 {
		t.Errorf("dy: got %v, want 60", got)
	}
}

func TestMouseState_ShiftPosInvalidLength(t *testing.T) {
	ms := MouseState{}

	ms.ShiftPos([]byte{1, 2, 3})
	ms.ShiftPos([]byte{1, 2, 3, 4, 5})

	if ms.dx.Load() != 0 || ms.dy.Load() != 0 {
		t.Error("invalid data should be ignored")
	}
}

func TestMouseState_Buttons(t *testing.T) {
	tests := []struct {
		name string
		data byte
		l    bool
		r    bool
		m    bool
	}{
		{name: "none", data: 0},
		{name: "left", data: 1, l: true},
		{name: "right", data: 2, r: true},
		{name: "middle", data: 4, m: true},
		{name: "left+right", data: 3, l: true, r: true},
		{name: "all", data: 7, l: true, r: true, m: true},
		{name: "left+middle", data: 5, l: true, m: true},
	}

	ms := MouseState{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms.SetButtons(test.data)
			l, r, m := ms.Buttons()
			if l != test.l || r != test.r || m != test.m {
				t.Errorf("got (%v, %v, %v), want (%v, %v, %v)", l, r, m, test.l, test.r, test.m)
			}
		})
	}
}

func TestMouseState_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	ms := MouseState{}
	events := 1000
	wg.Add(events * 3)

	for range events {
		go func() {
			data := make([]byte, 4)
			binary.BigEndian.PutUint16(data, uint16(rand.Int31n(100)-50))
			binary.BigEndian.PutUint16(data[2:], uint16(rand.Int31n(100)-50))
			ms.ShiftPos(data)
			wg.Done()
		}()
		go func() {
			ms.SetButtons(byte(rand.Intn(8)))
			wg.Done()
		}()
		go func() {
			ms.Buttons()
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestConstants(t *testing.T) {
	// MouseBtnState
	if MouseLeft != 1 || MouseRight != 2 || MouseMiddle != 4 {
		t.Error("invalid MouseBtnState constants")
	}

	// Device
	if RetroPad != 0 || Keyboard != 1 || Mouse != 2 {
		t.Error("invalid Device constants")
	}

	// Mouse events
	if MouseMove != 0 || MouseButton != 1 {
		t.Error("invalid mouse event constants")
	}

	// Limits
	if maxPort != 4 || numAxes != 4 || RetrokLast != 342 {
		t.Error("invalid limit constants")
	}
}
