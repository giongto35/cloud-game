package games

import (
	"reflect"
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
		library := NewLibrary(Config{
			BasePath:  test.directory,
			Supported: []string{"gba", "zip", "nes"},
			Ignored:   []string{"neogeo", "pgm"},
		})
		library.Scan()
		games := library.GetAll()

		list := _map(games, func(meta GameMetadata) string {
			return meta.Name
		})

		if !reflect.DeepEqual(test.expected, list) {
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
