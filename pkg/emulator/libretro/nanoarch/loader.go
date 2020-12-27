package nanoarch

import (
	"errors"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"unsafe"
)

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <dlfcn.h>
*/
import "C"

func open(file string) unsafe.Pointer {
	cs := C.CString(file)
	defer C.free(unsafe.Pointer(cs))
	return C.dlopen(cs, C.RTLD_LAZY)
}

func loadFunction(handle unsafe.Pointer, name string) unsafe.Pointer {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	pointer := C.dlsym(handle, cs)
	return pointer
}

func loadLib(filepath string) (handle unsafe.Pointer, err error) {
	handle = open(filepath)
	if handle == nil {
		e := C.dlerror()
		if e != nil {
			err = errors.New(C.GoString(e))
		} else {
			err = errors.New("couldn't load the lib")
		}
	}
	return
}

func loadLibRollingRollingRolling(filepath string) (handle unsafe.Pointer, err error) {
	dir, lib := path.Dir(filepath), path.Base(filepath)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.New("couldn't find 'n load the lib")
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), lib) {
			handle = open(path.Join(dir, file.Name()))
			if handle != nil {
				return handle, nil
			}
		}
	}
	return nil, errors.New("couldn't find 'n load the lib")
}

func closeLib(handle unsafe.Pointer) (err error) {
	if handle == nil {
		return
	}

	code := int(C.dlclose(handle))
	if code != 0 {
		return errors.New("couldn't close the lib (" + strconv.Itoa(code) + ")")
	}
	return
}
