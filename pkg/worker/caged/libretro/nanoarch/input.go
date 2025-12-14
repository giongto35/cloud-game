package nanoarch

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
)

/*
#include <stdint.h>
#include "libretro.h"

void input_cache_set_port(unsigned port, uint32_t buttons,
                          int16_t axis0, int16_t axis1, int16_t axis2, int16_t axis3);
void input_cache_set_keyboard_key(unsigned id, uint8_t pressed);
void input_cache_set_mouse(int16_t dx, int16_t dy, uint8_t buttons);
void input_cache_clear(void);
*/
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

// SyncToCache syncs the entire input state to the C-side cache.
// Call this once before each Run() instead of having C call back into Go.
func (s *InputState) SyncToCache() {
	for port := uint(0); port < maxPort; port++ {
		buttons := atomic.LoadUint32(&s[port].keys)
		axis0 := C.int16_t(atomic.LoadInt32(&s[port].axes[0]))
		axis1 := C.int16_t(atomic.LoadInt32(&s[port].axes[1]))
		axis2 := C.int16_t(atomic.LoadInt32(&s[port].axes[2]))
		axis3 := C.int16_t(atomic.LoadInt32(&s[port].axes[3]))
		C.input_cache_set_port(C.uint(port), C.uint32_t(buttons), axis0, axis1, axis2, axis3)
	}
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

// SyncToCache syncs keyboard state to the C-side cache.
func (ks *KeyboardState) SyncToCache() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for id, pressed := range ks.keys {
		C.input_cache_set_keyboard_key(C.uint(id), C.uint8_t(pressed))
	}
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
	dxy := binary.BigEndian.Uint32(data)
	ms.dx.Add(int32(int16(dxy >> 16)))
	ms.dy.Add(int32(int16(dxy)))
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

// SyncToCache syncs mouse state to the C-side cache.
// This consumes the delta values (swaps to 0).
func (ms *MouseState) SyncToCache() {
	dx := C.int16_t(ms.dx.Swap(0))
	dy := C.int16_t(ms.dy.Swap(0))
	buttons := C.uint8_t(ms.buttons.Load())
	C.input_cache_set_mouse(dx, dy, buttons)
}
