package games

import (
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

func TestLibraryScan(t *testing.T) {
	tests := []struct {
		directory string
		expected  []struct {
			name   string
			system string
		}
	}{
		{
			directory: "../../assets/games",
			expected: []struct {
				name   string
				system string
			}{
				{name: "Alwa's Awakening (Demo)", system: "nes"},
				{name: "Sushi The Cat", system: "gba"},
				{name: "anguna", system: "gba"},
			},
		},
	}

	emuConf := config.Emulator{Libretro: config.LibretroConfig{}}
	emuConf.Libretro.Cores.List = map[string]config.LibretroCoreConfig{
		"nes": {Roms: []string{"nes"}},
		"gba": {Roms: []string{"gba"}},
	}

	l := logger.NewConsole(false, "w", false)
	for _, test := range tests {
		library := NewLib(config.Library{
			BasePath:  test.directory,
			Supported: []string{"gba", "zip", "nes"},
		}, emuConf, l)
		library.Scan()
		games := library.GetAll()

		all := true
		for _, expect := range test.expected {
			found := false
			for _, game := range games {
				if game.Name == expect.name && (expect.system != "" && expect.system == game.System) {
					found = true
					break
				}
			}
			all = all && found
		}
		if !all {
			t.Errorf("Test fail for dir %v with %v != %v", test.directory, games, test.expected)
		}
	}
}

func Benchmark(b *testing.B) {
	log := logger.Default()
	logger.SetGlobalLevel(logger.Disabled)
	library := NewLib(config.Library{
		BasePath:  "../../assets/games",
		Supported: []string{"gba", "zip", "nes"},
	}, config.Emulator{}, log)

	for range b.N {
		library.Scan()
		_ = library.GetAll()
	}
}
