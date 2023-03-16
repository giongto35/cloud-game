package remotehttp

import (
	"reflect"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config/emulator"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		declared  []string
		installed []string
		diff      []string
	}{
		{},
		{
			declared:  []string{"a", "b"},
			installed: []string{},
			diff:      []string{"a", "b"},
		},
		{
			declared:  []string{},
			installed: []string{"c"},
		},
		{
			declared:  []string{"a", "b", "c"},
			installed: []string{"c", "a", "b"},
		},
		{
			declared:  []string{"a", "b", "c"},
			installed: []string{"c"},
			diff:      []string{"a", "b"},
		},
		{
			declared:  []string{"a", "b", "c", "c", "c", "a", "d"},
			installed: []string{"c", "c", "c", "a", "a", "a"},
			diff:      []string{"b", "d"},
		},
	}

	toCoreInfo := func(names []string) (r []emulator.CoreInfo) {
		for _, n := range names {
			r = append(r, emulator.CoreInfo{Name: n})
		}
		return
	}

	for _, test := range tests {
		difference := diff(toCoreInfo(test.declared), toCoreInfo(test.installed))
		if !reflect.DeepEqual(toCoreInfo(test.diff), difference) {
			t.Errorf("wrong diff for %v <- %v = %v != %v",
				test.declared, test.installed, test.diff, difference)
		}
	}
}
