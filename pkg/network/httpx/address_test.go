package httpx

import (
	"testing"
)

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		input Address
		host  string
		port  int
	}{
		{input: "", host: "", port: 0},
		{input: ":", host: "", port: 0},
		{input: "https://garbage.com:99a9a", host: "", port: 0},
		{input: ":9000", host: "", port: 9000},
		{input: "not-garbage:9999", host: "not-garbage", port: 9999},
		{input: "[::1]", host: "", port: 0},
		{input: ":90", host: "", port: 90},
		{input: "localhost:90", host: "localhost", port: 90},
		{input: "localhost:a8a", host: "localhost", port: 0},
	}

	for _, test := range tests {
		host, port := test.input.SplitHostPort()
		if host != test.host || port != test.port {
			t.Errorf(
				"Test fail for expected host %v port %v but got %v %v",
				test.host, test.port, host, port,
			)
		}
	}
}
