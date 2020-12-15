package downloader

import "testing"

func TestDownloader(t *testing.T) {
	down := NewGrabDownloader(struct{}{})
	down.Download(
		".",
		"https://github.com/giongto35/cloud-game/blob/master/assets/emulator/libretro/cores/citra_libretro.so?raw=true",
		"https://github.com/giongto35/cloud-game/blob/master/assets/emulator/libretro/cores/mgba_libretro.so?raw=true",
	)
}
