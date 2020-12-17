package core

import (
	"errors"
	"runtime"
)

// See: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63.
var libretroOsArchMap = map[string]ArchInfo{
	"linux:amd64":   {os: "linux", arch: "x86_64", LibExt: "so"},
	"linux:arm":     {os: "linux", arch: "armv7-neon-hf", LibExt: "armv7-neon-hf.so"},
	"windows:amd64": {os: "windows", arch: "x86_64", LibExt: "dll"},
	"darwin:amd64":  {os: "osx", arch: "x86_64", vendor: "apple", LibExt: "dylib"},
}

// ArchInfo contains Libretro core lib platform info.
// And cores are just C-compiled libraries.
// See: https://buildbot.libretro.com/nightly.
type ArchInfo struct {
	// bottom: x86_64, x86, ...
	arch string
	// middle: windows, ios, ...
	os string
	// top level: apple, nintendo, ...
	vendor string

	// platform dependent library file extension
	LibExt string
}

func GetCoreExt() (ArchInfo, error) {
	key := runtime.GOOS + ":" + runtime.GOARCH
	if arch, ok := libretroOsArchMap[key]; ok {
		return arch, nil
	} else {
		return ArchInfo{}, errors.New("core mapping not found for " + key)
	}
}
