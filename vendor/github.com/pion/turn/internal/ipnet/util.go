package ipnet

import (
	"fmt"
	"net"
)

// AddrIPPort extracts the IP and Port from a net.Addr
func AddrIPPort(a net.Addr) (net.IP, int, error) {
	aUDP, ok := a.(*net.UDPAddr)
	if !ok {
		return nil, 0, fmt.Errorf("failed to cast net.Addr to *net.UDPAddr")
	}

	return aUDP.IP, aUDP.Port, nil
}

// AddrEqual asserts that two net.Addrs are equal
// Currently only supprots UDP but will be extended in the future to support others
func AddrEqual(a, b net.Addr) bool {
	aUDP, ok := a.(*net.UDPAddr)
	if !ok {
		return false
	}

	bUDP, ok := b.(*net.UDPAddr)
	if !ok {
		return false
	}

	return aUDP.IP.Equal(bUDP.IP) && aUDP.Port == bUDP.Port
}
