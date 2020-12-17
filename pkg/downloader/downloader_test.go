package downloader

import (
	"path/filepath"
	"testing"
)

func TestDownloader(t *testing.T) {
	downloader := NewDefaultDownloader()
	path, _ := filepath.Abs(".")

	emus := []string{
		"mgba_libretro", "pcsx_rearmed_libretro", "nestopia_libretro",
		"snes9x_libretro", "fbneo_libretro", "mupen64plus_next_libretro",
	}

	urls := []string{
		"https://github.com/giongto35/cloud-game/blob/master/assets/emulator/libretro/cores/citra_libretro.so?raw=true"}

	for _, e := range emus {
		urls = append(urls, "https://buildbot.libretro.com/nightly/windows/x86_64/latest/"+
			e+".dll.zip")
	}

	downloader.Download(path, urls...)
}
