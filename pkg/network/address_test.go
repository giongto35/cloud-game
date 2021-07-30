package network

import (
	"testing"
)

func TestAddressPort(t *testing.T) {
	tests := []struct {
		input Address
		port  int
		err   string
	}{
		{input: "", port: 0, err: "no address"},
		{input: ":", port: 0, err: "port is not a number"},
		{input: "https://garbage.com:99a9a", port: 0, err: "port is not a number"},
		{input: ":9000", port: 9000},
		{input: "not-garbage:9999", port: 9999},
	}

	for _, test := range tests {
		port, err := test.input.Port()
		if port != test.port || (err != nil && test.err != err.Error()) {
			t.Errorf("Test fail for expected port %v but got %v with error %v", test.port, port, err)
		}
	}
}
