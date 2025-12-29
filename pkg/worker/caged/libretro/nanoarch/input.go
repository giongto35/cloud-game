package nanoarch

import (
	"encoding/binary"
	"sync/atomic"
)

/*
#include <stdint.h>
#include "libretro.h"

void input_cache_set_port(unsigned port, uint32_t buttons,
                          int16_t lx, int16_t ly, int16_t rx, int16_t ry,
                          int16_t l2, int16_t r2);
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
//   - int16 analog axes x4 (left stick, right stick)
//   - int16 analog triggers x2 (L2, R2)
type InputState [maxPort]struct {
	keys     uint32 // lower 16 bits used
	axes     int64  // packed: [LX:16][LY:16][RX:16][RY:16]
	triggers int32  // packed: [L2:16][R2:16]
}

// SetInput sets input state for a player.
//
//	[BTN:2][LX:2][LY:2][RX:2][RY:2][L2:2][R2:2]
func (s *InputState) SetInput(port int, data []byte) {
	if len(data) < 2 {
		return
	}

	// Buttons
	atomic.StoreUint32(&s[port].keys, uint32(binary.LittleEndian.Uint16(data)))

	// Axes - pack into int64
	var packedAxes int64
	for i := 0; i < numAxes && i*2+3 < len(data); i++ {
		axis := int64(int16(binary.LittleEndian.Uint16(data[i*2+2:])))
		packedAxes |= (axis & 0xFFFF) << (i * 16)
	}
	atomic.StoreInt64(&s[port].axes, packedAxes)

	// Analog triggers L2, R2 - pack into int32
	if len(data) >= 14 {
		l2 := int32(int16(binary.LittleEndian.Uint16(data[10:])))
		r2 := int32(int16(binary.LittleEndian.Uint16(data[12:])))
		atomic.StoreInt32(&s[port].triggers, (l2&0xFFFF)|((r2&0xFFFF)<<16))
	}
}

// SyncToCache syncs input state to C-side cache before Run().
func (s *InputState) SyncToCache() {
	for p := uint(0); p < maxPort; p++ {
		keys := atomic.LoadUint32(&s[p].keys)
		axes := atomic.LoadInt64(&s[p].axes)
		triggers := atomic.LoadInt32(&s[p].triggers)

		C.input_cache_set_port(C.uint(p), C.uint32_t(keys),
			C.int16_t(axes),
			C.int16_t(axes>>16),
			C.int16_t(axes>>32),
			C.int16_t(axes>>48),
			C.int16_t(triggers),
			C.int16_t(triggers>>16))
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
