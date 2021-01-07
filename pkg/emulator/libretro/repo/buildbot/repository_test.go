package buildbot

import (
	"testing"

	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
)

func TestBuildbotRepo(t *testing.T) {
	testAddress := "http://test.me"
	tests := []struct {
		file        string
		compression string
		arch        core.ArchInfo
		resultUrl   string
	}{
		{
			file: "uber_core",
			arch: core.ArchInfo{
				Os:     "linux",
				Arch:   "x86_64",
				LibExt: ".so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so",
		},
		{
			file:        "uber_core",
			compression: "zip",
			arch: core.ArchInfo{
				Os:     "linux",
				Arch:   "x86_64",
				LibExt: ".so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so.zip",
		},
		{
			file: "uber_core",
			arch: core.ArchInfo{
				Os:     "osx",
				Arch:   "x86_64",
				Vendor: "apple",
				LibExt: ".dylib",
			},
			resultUrl: testAddress + "/" + "apple/osx/x86_64/latest/uber_core.dylib",
		},
	}

	for _, test := range tests {
		repo := NewBuildbotRepo(testAddress, test.compression)
		url := repo.GetCoreUrl(test.file, test.arch)
		if url != test.resultUrl {
			t.Errorf("seems that expected link address is incorrect (%v) for file %s %+v",
				url, test.file, test.arch)
		}
	}
}
