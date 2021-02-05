package nanoarch

/*
#include "libretro.h"
#include <stdlib.h>

size_t bridge_retro_get_memory_size(void *f, unsigned id);
void* bridge_retro_get_memory_data(void *f, unsigned id);
bool bridge_retro_serialize(void *f, void *data, size_t size);
bool bridge_retro_unserialize(void *f, void *data, size_t size);
size_t bridge_retro_serialize_size(void *f);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// !global emulator lib state
var (
	retroGetMemoryData unsafe.Pointer
	retroGetMemorySize unsafe.Pointer
	retroSerialize     unsafe.Pointer
	retroSerializeSize unsafe.Pointer
	retroUnserialize   unsafe.Pointer
)

// defines any memory state of the emulator
type state []byte

type mem struct {
	ptr  unsafe.Pointer
	size uint
}

// saveStateSize returns the amount of data the implementation requires
// to serialize internal state (save states).
func saveStateSize() uint { return uint(C.bridge_retro_serialize_size(retroSerializeSize)) }

// getSaveState returns emulator internal state.
func getSaveState() (state, error) {
	size := saveStateSize()
	data := C.malloc(C.size_t(size))
	defer C.free(data)
	if !bool(C.bridge_retro_serialize(retroSerialize, data, C.size_t(size))) {
		return nil, errors.New("retro_serialize failed")
	}
	return C.GoBytes(data, C.int(size)), nil
}

// restoreSaveState restores emulator internal state.
func restoreSaveState(st state) error {
	if len(st) == 0 {
		return nil
	}
	size := saveStateSize()
	if !bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&st[0]), C.size_t(size))) {
		return errors.New("retro_unserialize failed")
	}
	return nil
}

// getSaveRAM returns the game save RAM (cartridge) data or a nil slice.
func getSaveRAM() state {
	mem := ptSaveRAM()
	if mem == nil {
		return nil
	}
	return C.GoBytes(mem.ptr, C.int(mem.size))
}

// restoreSaveRAM restores game save RAM.
func restoreSaveRAM(st state) {
	if len(st) == 0 {
		return
	}
	if mem := ptSaveRAM(); mem != nil {
		sram := (*[1 << 30]byte)(mem.ptr)[:mem.size:mem.size]
		copy(sram, st)
	}
}

// getMemorySize returns memory region size.
func getMemorySize(id uint) uint {
	return uint(C.bridge_retro_get_memory_size(retroGetMemorySize, C.uint(id)))
}

// getMemoryData returns a pointer to memory data.
func getMemoryData(id uint) unsafe.Pointer {
	return C.bridge_retro_get_memory_data(retroGetMemoryData, C.uint(id))
}

// ptSaveRam return SRAM memory pointer if core supports it or nil.
func ptSaveRAM() *mem {
	ptr, size := getMemoryData(C.RETRO_MEMORY_SAVE_RAM), getMemorySize(C.RETRO_MEMORY_SAVE_RAM)
	if ptr == nil || size == 0 {
		return nil
	}
	return &mem{ptr: ptr, size: size}
}
