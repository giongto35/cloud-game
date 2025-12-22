package games

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestAliasFileMaybe(t *testing.T) {
	lib := &library{
		config: libConf{
			aliasFile: "alias",
			path:      os.TempDir(),
		},
		log: logger.NewConsole(false, "w", false),
	}

	contents := "a=b\nc=d\n"

	path := filepath.Join(lib.config.path, lib.config.aliasFile)
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.RemoveAll(path); err != nil {
			t.Error(err)
		}
	}()

	want := map[string]string{}
	want["a"] = "b"
	want["c"] = "d"

	aliases := lib.AliasFileMaybe()

	if !reflect.DeepEqual(aliases, want) {
		t.Errorf("AliasFileMaybe() = %v, want %v", aliases, want)
	}
}

func TestAliasFileMaybeNot(t *testing.T) {
	lib := &library{
		config: libConf{
			path: os.TempDir(),
		},
		log: logger.NewConsole(false, "w", false),
	}

	aliases := lib.AliasFileMaybe()
	if aliases != nil {
		t.Errorf("should be nil, but %v", aliases)
	}
}

func Benchmark(b *testing.B) {
	log := logger.Default()
	logger.SetGlobalLevel(logger.Disabled)
	library := NewLib(config.Library{
		BasePath:  "../../assets/games",
		Supported: []string{"gba", "zip", "nes"},
	}, config.Emulator{}, log)

	for b.Loop() {
		library.Scan()
		_ = library.GetAll()
	}
}
