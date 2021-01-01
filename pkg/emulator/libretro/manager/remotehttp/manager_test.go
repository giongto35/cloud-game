package remotehttp

import (
	"reflect"
	"testing"
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

	for _, test := range tests {
		difference := diff(test.declared, test.installed)
		if !reflect.DeepEqual(test.diff, difference) {
			t.Errorf("wrong diff for %v <- %v = %v != %v",
				test.declared, test.installed, test.diff, difference)
		}
	}
}
