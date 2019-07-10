package nanoarch

/*
#include "libretro.h"
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <dlfcn.h>
#include <string.h>

size_t bridge_retro_get_memory_size(void *f, unsigned id);
void* bridge_retro_get_memory_data(void *f, unsigned id);
*/
import "C"

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"unsafe"

	"github.com/giongto35/cloud-game/util"
)

var mutex sync.Mutex

const (
	memorySaveRAM = uint32(C.RETRO_MEMORY_SAVE_RAM)
)

// path returns the path of the SRAM file for the current core
//func path() string {
//return filepath.Join("", name())
//}

// GetMemorySize returns the size of a region of the memory.
// See memory constants.
func getMemorySize(id uint32) uint {
	return uint(C.bridge_retro_get_memory_size(retroGetMemorySize, C.unsigned(id)))
}

// GetMemoryData returns the size of a region of the memory.
// See memory constants.
func getMemoryData(id uint32) unsafe.Pointer {
	return C.bridge_retro_get_memory_data(retroGetMemoryData, C.unsigned(id))
}

// SaveSRAM saves the game SRAM to the filesystem
func (na *naEmulator) SaveSRAM() error {
	mutex.Lock()
	defer mutex.Unlock()

	//TODO: Check corer running
	//if !state.Global.CoreRunning {
	//return errors.New("core not running")
	//}

	len := getMemorySize(memorySaveRAM)
	ptr := getMemoryData(memorySaveRAM)
	if ptr == nil || len == 0 {
		return errors.New("unable to get SRAM address")
	}

	// convert the C array to a go slice
	bytes := C.GoBytes(ptr, C.int(len))
	//err := os.MkdirAll(settings.Current.SavefilesDirectory, os.ModePerm)
	//if err != nil {
	//return err
	//}

	fd, err := os.Create(util.GetSavePath(na.roomID))
	if err != nil {
		log.Println("Err: Cannot create path", err)
		return err
	}
	defer fd.Close()
	fd.Write(bytes)

	return nil
}

// LoadSRAM saves the game SRAM to the filesystem
func (na *naEmulator) LoadSRAM() error {
	mutex.Lock()
	defer mutex.Unlock()

	//if !state.Global.CoreRunning {
	//return errors.New("core not running")
	//}

	fd, err := os.Open(util.GetSavePath(na.roomID))
	if err != nil {
		log.Println("Err: Cannot get save path", err)
		return err
	}
	defer fd.Close()

	len := getMemorySize(memorySaveRAM)
	ptr := getMemoryData(memorySaveRAM)
	if ptr == nil || len == 0 {
		return errors.New("unable to get SRAM address")
	}

	// this *[1 << 30]byte points to the same memory as ptr, allowing to
	// overwrite this memory
	destination := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:len:len]
	source, err := ioutil.ReadAll(fd)
	if err != nil {
		log.Println("Err: source", err)
		return err
	}
	copy(destination, source)

	log.Println("Load SRAM successfully")
	return nil
}
