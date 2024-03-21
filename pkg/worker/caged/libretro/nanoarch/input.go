package nanoarch

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
)

//#include <stdint.h>
//#include "libretro.h"
import "C"

const (
	Released C.int16_t = iota
	Pressed
)

const RetrokLast = int(C.RETROK_LAST)

// InputState stores full controller state.
// It consists of:
//   - uint16 button values
//   - int16 analog stick values
type InputState [maxPort]RetroPadState

type (
	RetroPadState struct {
		keys uint32
		axes [dpadAxes]int32
	}
	KeyboardState struct {
		keys [RetrokLast]byte
		mod  uint16
		mu   sync.Mutex
	}
	MouseState struct {
		dx, dy  atomic.Int32
		buttons atomic.Int32
	}
)

type MouseBtnState int32

type Device byte

const (
	RetroPad Device = iota
	Keyboard
	Mouse
)

const (
	MouseMove = iota
	MouseButton
)

const (
	MouseLeft MouseBtnState = 1 << iota
	MouseRight
	MouseMiddle
)

const (
	maxPort  = 4
	dpadAxes = 4
)

// Input sets input state for some player in a game session.
func (s *InputState) Input(port int, data []byte) {
	atomic.StoreUint32(&s[port].keys, uint32(uint16(data[1])<<8+uint16(data[0])))
	for i, axes := 0, len(data); i < dpadAxes && i<<1+3 < axes; i++ {
		axis := i<<1 + 2
		atomic.StoreInt32(&s[port].axes[i], int32(data[axis+1])<<8+int32(data[axis]))
	}
}

// IsKeyPressed checks if some button is pressed by any player.
func (s *InputState) IsKeyPressed(port uint, key int) C.int16_t {
	return C.int16_t((atomic.LoadUint32(&s[port].keys) >> uint(key)) & 1)
}

// IsDpadTouched checks if D-pad is used by any player.
func (s *InputState) IsDpadTouched(port uint, axis uint) (shift C.int16_t) {
	return C.int16_t(atomic.LoadInt32(&s[port].axes[axis]))
}

// SetKey sets keyboard state.
//
//	0 1 2 3 4 5 6
//	[ KEY ] P MOD
//
//	KEY contains Libretro code of the keyboard key (4 bytes).
//	P contains 0 or 1 if the key is pressed (1 byte).
//	MOD contains bitmask for Alt | Ctrl | Meta | Shift keys press state (2 bytes).
//
// Returns decoded state from the input bytes.
func (ks *KeyboardState) SetKey(data []byte) (pressed bool, key uint, mod uint16) {
	if len(data) != 7 {
		return
	}

	press := data[4]
	pressed = press == 1
	key = uint(binary.BigEndian.Uint32(data))
	mod = binary.BigEndian.Uint16(data[5:])
	ks.mu.Lock()
	ks.keys[key] = press
	ks.mod = mod
	ks.mu.Unlock()
	return
}

func (ks *KeyboardState) Pressed(key uint) C.int16_t {
	ks.mu.Lock()
	press := ks.keys[key]
	ks.mu.Unlock()
	if press == 1 {
		return Pressed
	}
	return Released
}

// ShiftPos sets mouse relative position state.
//
//	0  1 2  3
//	[dx] [dy]
//
//	dx and dy are relative mouse coordinates
func (ms *MouseState) ShiftPos(data []byte) {
	if len(data) != 4 {
		return
	}
	dx := int16(data[0])<<8 + int16(data[1])
	dy := int16(data[2])<<8 + int16(data[3])
	ms.dx.Add(int32(dx))
	ms.dy.Add(int32(dy))
}

func (ms *MouseState) PopX() C.int16_t { return C.int16_t(ms.dx.Swap(0)) }
func (ms *MouseState) PopY() C.int16_t { return C.int16_t(ms.dy.Swap(0)) }

// SetButtons sets the state MouseBtnState of mouse buttons.
func (ms *MouseState) SetButtons(data byte) { ms.buttons.Store(int32(data)) }

func (ms *MouseState) Buttons() (l, r, m bool) {
	mbs := MouseBtnState(ms.buttons.Load())
	l = mbs&MouseLeft != 0
	r = mbs&MouseRight != 0
	m = mbs&MouseMiddle != 0
	return
}
