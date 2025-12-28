package nanoarch

import (
	"encoding/binary"
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
	maxPort    = 4
	numAxes    = 4
	RetrokLast = int(C.RETROK_LAST)
)

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

type MouseBtnState int32

const (
	MouseLeft MouseBtnState = 1 << iota
	MouseRight
	MouseMiddle
)

// InputState stores controller state for all ports.
//   - uint16 button bitmask
//   - int16 analog axes x4
type InputState [maxPort]struct {
	keys uint32
	axes [numAxes]int32
}

// SetInput sets input state for a player.
//
//	[BTN:2][AX0:2][AX1:2][AX2:2][AX3:2]
func (s *InputState) SetInput(port int, data []byte) {
	atomic.StoreUint32(&s[port].keys, uint32(binary.LittleEndian.Uint16(data)))
	for i := 0; i < numAxes && i*2+3 < len(data); i++ {
		atomic.StoreInt32(&s[port].axes[i], int32(int16(binary.LittleEndian.Uint16(data[i*2+2:]))))
	}
}

// Button check
func (s *InputState) Button(port, key uint) C.int16_t {
	return C.int16_t((atomic.LoadUint32(&s[port].keys) >> key) & 1)
}

// SyncToCache syncs input state to C-side cache before Run().
func (s *InputState) SyncToCache() {
	for p := uint(0); p < maxPort; p++ {
		a := &s[p].axes
		C.input_cache_set_port(C.uint(p), C.uint32_t(atomic.LoadUint32(&s[p].keys)),
			C.int16_t(atomic.LoadInt32(&a[0])), C.int16_t(atomic.LoadInt32(&a[1])),
			C.int16_t(atomic.LoadInt32(&a[2])), C.int16_t(atomic.LoadInt32(&a[3])))
	}
}

// KeyboardState tracks keys of the keyboard.
type KeyboardState struct {
	keys [6]atomic.Uint64 // 342 keys packed into 6 uint64s (384 bits)
	mod  atomic.Uint32
}

// SetKey sets keyboard state.
//
//	[KEY:4][P:1][MOD:2]
//
//	KEY - Libretro key code, P - pressed (0/1), MOD - modifier bitmask
func (ks *KeyboardState) SetKey(data []byte) (pressed bool, key uint, mod uint16) {
	if len(data) != 7 {
		return
	}
	key = uint(binary.BigEndian.Uint32(data))
	mod = binary.BigEndian.Uint16(data[5:])
	pressed = data[4] == 1

	idx, bit := key/64, uint64(1)<<(key%64)
	if pressed {
		ks.keys[idx].Or(bit)
	} else {
		ks.keys[idx].And(^bit)
	}
	ks.mod.Store(uint32(mod))

	return
}

// SyncToCache syncs keyboard state to C-side cache.
func (ks *KeyboardState) SyncToCache() {
	for id := 0; id < RetrokLast; id++ {
		pressed := (ks.keys[id/64].Load() >> (id % 64)) & 1
		C.input_cache_set_keyboard_key(C.uint(id), C.uint8_t(pressed))
	}
}

// MouseState tracks mouse delta and buttons.
type MouseState struct {
	dx, dy  atomic.Int32
	buttons atomic.Int32
}

// ShiftPos adds relative mouse movement.
//
//	[dx:2][dy:2]
func (ms *MouseState) ShiftPos(data []byte) {
	if len(data) != 4 {
		return
	}
	ms.dx.Add(int32(int16(binary.BigEndian.Uint16(data[:2]))))
	ms.dy.Add(int32(int16(binary.BigEndian.Uint16(data[2:]))))
}

func (ms *MouseState) SetButtons(b byte) { ms.buttons.Store(int32(b)) }

func (ms *MouseState) Buttons() (l, r, m bool) {
	b := MouseBtnState(ms.buttons.Load())
	return b&MouseLeft != 0, b&MouseRight != 0, b&MouseMiddle != 0
}

// SyncToCache syncs mouse state to C-side cache, consuming deltas.
func (ms *MouseState) SyncToCache() {
	C.input_cache_set_mouse(C.int16_t(ms.dx.Swap(0)), C.int16_t(ms.dy.Swap(0)), C.uint8_t(ms.buttons.Load()))
}
