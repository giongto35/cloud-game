package arch

import (
	"errors"
	"runtime"
)

// See: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63.
var libretroOsArchMap = map[string]Info{
	"linux:amd64":   {Os: "linux", Arch: "x86_64", LibExt: ".so"},
	"linux:arm":     {Os: "linux", Arch: "armv7-neon-hf", LibExt: ".so"},
	"windows:amd64": {Os: "windows", Arch: "x86_64", LibExt: ".dll"},
	"darwin:amd64":  {Os: "osx", Arch: "x86_64", Vendor: "apple", LibExt: ".dylib"},
	"darwin:arm64":  {Os: "osx", Arch: "arm64", Vendor: "apple", LibExt: ".dylib"},
}

// Info contains Libretro core lib platform info.
// And cores are just C-compiled libraries.
// See: https://buildbot.libretro.com/nightly.
type Info struct {
	// bottom: x86_64, x86, ...
	Arch string
	// middle: windows, ios, ...
	Os string
	// top level: apple, nintendo, ...
	Vendor string

	// platform dependent library file extension (dot-prefixed)
	LibExt string
}

func Guess() (Info, error) {
	key := runtime.GOOS + ":" + runtime.GOARCH
	if arch, ok := libretroOsArchMap[key]; ok {
		return arch, nil
	} else {
		return Info{}, errors.New("core mapping not found for " + key)
	}
}
