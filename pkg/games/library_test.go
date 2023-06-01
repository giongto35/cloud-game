package games

import (
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

func TestLibraryScan(t *testing.T) {
	tests := []struct {
		directory string
		expected  []string
	}{
		{
			directory: "../../assets/games",
			expected: []string{
				"Alwa's Awakening (Demo)", "Sushi The Cat", "anguna",
			},
		},
	}

	l := logger.NewConsole(false, "w", false)
	for _, test := range tests {
		library := NewLib(config.Library{
			BasePath:  test.directory,
			Supported: []string{"gba", "zip", "nes"},
		}, l)
		library.Scan()
		games := library.GetAll()

		list := _map(games, func(g GameMetadata) string { return g.Name })

		all := true
		for _, expect := range test.expected {
			found := false
			for _, game := range list {
				if game == expect {
					found = true
					break
				}
			}
			all = all && found
		}
		if !all {
			t.Errorf("Test fail for dir %v with %v != %v", test.directory, list, test.expected)
		}
	}
}

func Benchmark(b *testing.B) {
	log := logger.Default()
	logger.SetGlobalLevel(logger.Disabled)
	library := NewLib(config.Library{
		BasePath:  "../../assets/games",
		Supported: []string{"gba", "zip", "nes"},
	}, log)

	for i := 0; i < b.N; i++ {
		library.Scan()
		_ = library.GetAll()
	}
}

func _map(vs []GameMetadata, f func(info GameMetadata) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
