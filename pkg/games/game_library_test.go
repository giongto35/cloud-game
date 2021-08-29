package games

import (
	"testing"
)

func TestLibraryScan(t *testing.T) {
	tests := []struct {
		directory string
		expected  []string
	}{
		{
			directory: "../../assets/games",
			expected: []string{
				"Super Mario Bros", "Sushi The Cat", "anguna",
			},
		},
	}

	for _, test := range tests {
		library := NewLib(Config{
			BasePath:  test.directory,
			Supported: []string{"gba", "zip", "nes"},
			Ignored:   []string{"neogeo", "pgm"},
		})
		library.Scan()
		games := library.GetAll()

		list := _map(games, func(meta GameMetadata) string {
			return meta.Name
		})

		// ^2 complexity (;
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

func _map(vs []GameMetadata, f func(info GameMetadata) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
