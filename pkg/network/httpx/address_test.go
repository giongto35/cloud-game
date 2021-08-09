package httpx

import (
	"net"
	"testing"
)

type testListener struct {
	addr net.TCPAddr
}

func (tl testListener) Accept() (net.Conn, error) { return nil, nil }
func (tl testListener) Close() error              { return nil }
func (tl testListener) Addr() net.Addr            { return &tl.addr }

func NewTCP(port int) Listener {
	return Listener{testListener{addr: net.TCPAddr{Port: port}}}
}

func TestMergeAddresses(t *testing.T) {
	tests := []struct {
		addr string
		zone string
		ls   Listener
		rez  string
	}{
		{addr: "", rez: "localhost"},
		{addr: ":", ls: NewTCP(0), rez: "localhost"},
		{addr: "", ls: NewTCP(393), rez: "localhost:393"},
		{addr: ":8080", ls: NewTCP(8080), rez: "localhost:8080"},
		{addr: ":8080", ls: NewTCP(8081), rez: "localhost:8081"},
		{addr: "host:8080", ls: NewTCP(8080), rez: "host:8080"},
		{addr: "host:8080", ls: NewTCP(8081), rez: "host:8081"},
		{addr: "host:8080", zone: "test", ls: NewTCP(8081), rez: "test.host:8081"},
		{addr: ":80", ls: NewTCP(80), rez: "localhost"},
		{addr: ":", ls: NewTCP(344), rez: "localhost:344"},
		{addr: "https://garbage.com:99a9a", rez: "https://garbage.com:99a9a"},
		{addr: "[::]", rez: "[::]"},
	}

	for _, test := range tests {
		address := buildAddress(test.addr, test.zone, test.ls)
		if address != test.rez {
			t.Errorf("expected %v, got %v", test.rez, address)
		}
	}
}
