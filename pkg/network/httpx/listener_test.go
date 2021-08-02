package httpx

import (
	"net"
	"strings"
	"testing"
)

func TestListenerCreation(t *testing.T) {
	tests := []struct {
		addr     string
		endpoint string
		port     int
		random   bool
		error    bool
	}{
		{addr: ":80", endpoint: "0.0.0.0:80", port: 80},
		{addr: ":", endpoint: "0.0.0.0", random: true},
		{addr: ":0", endpoint: "0.0.0.0", random: true},
		{addr: "", endpoint: "0.0.0.0", random: true},
		{addr: "https://garbage.com:99a9a", error: true},
		{addr: ":8082", endpoint: "0.0.0.0:8082", port: 8082},
		{addr: "localhost:8888", endpoint: "127.0.0.1:8888", port: 8888},
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

		addr := ls.Addr().(*net.TCPAddr)

		hasPort := addr.Port > 0
		isPortSame := addr.Port == test.port
		isAddrSame := strings.HasPrefix(addr.String(), test.endpoint)
		isEndpointSame := addr.String() == test.endpoint

		if test.random {
			if !hasPort && !isAddrSame {
				t.Errorf("expected the same addr %v with a random port, got %v %v", test.endpoint, addr.IP, addr.Port)
			}
			_ = ls.Close()
			continue
		}

		if !isPortSame {
			t.Errorf("expected the same port %v != %v", test.port, addr.Port)
		} else if !isEndpointSame {
			t.Errorf("expected the same full address %v != %v", test.endpoint, addr.String())
		}
		_ = ls.Close()
	}
}

func TestFailOnPortInUse(t *testing.T) {
	a, err := NewListener("127.0.0.1:3333", false)
	defer func(a *Listener) {
		if a != nil {
			_ = a.Close()
		}
	}(a)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	b, err := NewListener("127.0.0.1:3333", false)
	defer func(a *Listener) {
		if b != nil {
			_ = b.Close()
		}
	}(b)
	if err == nil {
		t.Errorf("expected busy port error, but got none")
	}
}

func TestListenerPortRoll(t *testing.T) {
	a, err := NewListener("127.0.0.1:3333", false)
	defer func(a *Listener) {
		if a != nil {
			_ = a.Close()
		}
	}(a)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	b, err := NewListener("127.0.0.1:3333", true)
	defer func(a *Listener) {
		if b != nil {
			_ = b.Close()
		}
	}(b)
	if err == nil {
		t.Errorf("expected busy port error, but got none")
	}
}
