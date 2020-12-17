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
				LibExt: "so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so",
		},
		{
			file:        "uber_core",
			compression: "zip",
			arch: core.ArchInfo{
				Os:     "linux",
				Arch:   "x86_64",
				LibExt: "so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so.zip",
		},
		{
			file: "uber_core",
			arch: core.ArchInfo{
				Os:     "osx",
				Arch:   "x86_64",
				Vendor: "apple",
				LibExt: "dylib",
			},
			resultUrl: testAddress + "/" + "apple/osx/x86_64/latest/uber_core.dylib",
		},
	}

	for _, test := range tests {
		repo := NewBuildbotRepo(testAddress).WithCompression(test.compression)
		data := repo.GetCoreData(test.file, test.arch)
		if data.Url != test.resultUrl {
			t.Errorf("seems that expected link address is incorrect for file %s %+v", test.file, test.arch)
		}
	}
}
