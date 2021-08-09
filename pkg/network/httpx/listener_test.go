package httpx

import (
	"net"
	"strings"
	"testing"
)

func TestListenerCreation(t *testing.T) {
	tests := []struct {
		addr   string
		port   string
		random bool
		error  bool
	}{
		{addr: ":80", port: "80"},
		{addr: ":", random: true},
		{addr: ":0", random: true},
		{addr: "", random: true},
		{addr: "https://garbage.com:99a9a", error: true},
		{addr: ":8082", port: "8082"},
		{addr: "localhost:8888", port: "8888"},
		{addr: "localhost:abc1", error: true},
	}

	for _, test := range tests {
		ls, err := NewListener(test.addr, false)

		if test.error {
			if err == nil {
				t.Errorf("expected error, but got none")
			}
			continue
		}

		if !test.error && err != nil {
			t.Errorf("unexpected error %v", err)
			continue
		}

		defer ls.Close()

		addr := ls.Addr().(*net.TCPAddr)
		port := ls.GetPort()

		hasPort := port > 0
		isPortSame := strings.HasSuffix(addr.String(), ":"+test.port)

		if test.random {
			if !hasPort {
				t.Errorf("expected a random port, got %v", port)
			}
			continue
		}

		if !isPortSame {
			t.Errorf("expected the same port %v != %v", test.port, port)
		}
	}
}

func TestFailOnPortInUse(t *testing.T) {
	a, err := NewListener(":3333", false)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer a.Close()
	_, err = NewListener(":3333", false)
	if err == nil {
		t.Errorf("expected busy port error, but got none")
	}
}

func TestListenerPortRoll(t *testing.T) {
	a, err := NewListener("127.0.0.1:3333", false)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer a.Close()
	b, err := NewListener("127.0.0.1:3333", true)
	if err != nil {
		t.Errorf("expected no port error, but got %v", err)
	}
	b.Close()
}
