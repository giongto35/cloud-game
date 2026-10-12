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

func TestSnesMultitapOnlyForBomberman(t *testing.T) {
	var conf WorkerConfig
	if _, err := LoadConfig(&conf, ""); err != nil {
		t.Fatal(err)
	}
	snes := conf.Emulator.GetLibretroCoreConfig("snes")
	if len(snes.Hid) != 0 {
		t.Fatalf("SNES must use plain joypads by default (multitap breaks 2P games), got %v", snes.Hid)
	}
	if got := snes.Hid4rom["bomberman"][1]; len(got) != 1 || got[0] != 257 {
		t.Fatalf("Bomberman games must get the multitap on port 2, got %v", snes.Hid4rom)
	}
}
