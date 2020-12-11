package core

import (
	"errors"
	"runtime"
)

// see: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
var libretroOsArchMap = map[string]ArchInfo{
	"linux:amd64":   {os: "linux", arch: "x86_64", Lib: ".so"},
	"linux:arm":     {os: "linux", arch: "armv7-neon-hf", Lib: ".armv7-neon-hf.so"},
	"windows:amd64": {os: "windows", arch: "x86_64", Lib: ".dll"},
	"darwin:amd64":  {os: "osx", arch: "x86_64", vendor: "apple", Lib: ".dylib"},
}

type ArchInfo struct {
	os     string
	arch   string
	vendor string
	Lib    string
}

func GetCoreExt() (ArchInfo, error) {
	key := runtime.GOOS + ":" + runtime.GOARCH
	if arch, ok := libretroOsArchMap[key]; ok {
		return arch, nil
	} else {
		return ArchInfo{}, errors.New("core mapping not found for " + key)
	}
}
