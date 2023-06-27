package config

import "testing"

func TestGetEmulator(t *testing.T) {
	tests := []struct {
		rom      string
		path     string
		config   map[string]LibretroCoreConfig
		emulator string
	}{
		{
			rom:  "nes",
			path: "test/game.nes",
			config: map[string]LibretroCoreConfig{
				"snes": {Roms: []string{"nes"}},
				"nes":  {Folder: "test", Roms: []string{"nes"}},
			},
			emulator: "nes",
		},
		{
			rom:  "nes",
			path: "nes/game.nes",
			config: map[string]LibretroCoreConfig{
				"snes": {Roms: []string{"nes"}},
				"nes":  {Roms: []string{"nes"}},
			},
			emulator: "nes",
		},
		{
			rom:  "nes",
			path: "test2/game.nes",
			config: map[string]LibretroCoreConfig{
				"snes": {Roms: []string{"snes"}},
				"nes":  {Roms: []string{"nes"}},
			},
			emulator: "nes",
		},
	}

	for _, test := range tests {
		emu := Emulator{Libretro: LibretroConfig{}}
		emu.Libretro.Cores.List = test.config
		em := emu.GetEmulator(test.rom, test.path)
		if test.emulator != em {
			t.Errorf("expected result: %v, but was %v with: %v, %v", test.emulator, em, test.rom, test.path)
		}
	}
}
