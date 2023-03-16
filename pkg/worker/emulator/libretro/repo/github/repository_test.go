package github

import (
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro"
	"testing"
)

func TestBuildbotRepo(t *testing.T) {
	testAddress := "https://test.me"
	tests := []struct {
		file        string
		compression string
		arch        libretro.ArchInfo
		resultUrl   string
	}{
		{
			file: "uber_core",
			arch: libretro.ArchInfo{
				Os:     "linux",
				Arch:   "x86_64",
				LibExt: ".so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so?raw=true",
		},
		{
			file:        "uber_core",
			compression: "zip",
			arch: libretro.ArchInfo{
				Os:     "linux",
				Arch:   "x86_64",
				LibExt: ".so",
			},
			resultUrl: testAddress + "/" + "linux/x86_64/latest/uber_core.so.zip?raw=true",
		},
		{
			file: "uber_core",
			arch: libretro.ArchInfo{
				Os:     "osx",
				Arch:   "x86_64",
				Vendor: "apple",
				LibExt: ".dylib",
			},
			resultUrl: testAddress + "/" + "apple/osx/x86_64/latest/uber_core.dylib?raw=true",
		},
	}

	for _, test := range tests {
		repo := NewGithubRepo(testAddress, test.compression)
		url := repo.GetCoreUrl(test.file, test.arch)
		if url != test.resultUrl {
			t.Errorf("seems that expected link address is incorrect (%v) for file %s %+v",
				url, test.file, test.arch)
		}
	}
}
