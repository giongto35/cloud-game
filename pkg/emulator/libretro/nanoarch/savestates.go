// Package savestates takes care of serializing and unserializing the game RAM
// to the host filesystem.
package nanoarch

/*
#include "libretro.h"
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <dlfcn.h>
#include <string.h>

bool bridge_retro_serialize(void *f, void *data, size_t size);
bool bridge_retro_unserialize(void *f, void *data, size_t size);
size_t bridge_retro_serialize_size(void *f);
*/
import "C"

import (
	"io/ioutil"
	"sync"
)

var saveLock int32
var m sync.Mutex

func (na *naEmulator) GetLock() {
	//atomic.CompareAndSwapInt32(&saveLock, 0, 1)
	m.Lock()
}

func (na *naEmulator) ReleaseLock() {
	//atomic.CompareAndSwapInt32(&saveLock, 1, 0)
	m.Unlock()
}

// Save the current state to the filesystem. name is the name of the
// savestate file to save to, without extension.
func (na *naEmulator) Save() error {
	path := na.GetHashPath()

	na.GetLock()
	defer na.ReleaseLock()

	s := serializeSize()
	bytes, err := serialize(s)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, bytes, 0644)
}

// Load the state from the filesystem
func (na *naEmulator) Load() error {
	path := na.GetHashPath()

	na.GetLock()
	defer na.ReleaseLock()

	s := serializeSize()
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = unserialize(bytes, s)
	return err
}
