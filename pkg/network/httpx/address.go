package httpx

import (
	"net"
	"strconv"
)

// buildAddress joins network host from the first param,
// zone from the second, and
// the port value of a listener from the third param.
//
// As example, address host.com:8080 and listener 123.123.123.123:8888 will be
// transformed to host.com:8888.
func buildAddress(address string, zone string, l Listener) string {
	addr, _, err := net.SplitHostPort(address)
	if err != nil {
		addr = address
	}
	if addr == "" {
		addr = "localhost"
	}

	port := l.GetPort()
	if port > 0 && port != 80 && port != 443 {
		addr += ":" + strconv.Itoa(port)
	}

	return withZonePrefix(addr, zone)
}

func withZonePrefix(host string, zone string) string {
	if zone != "" {
		return zone + "." + host
	}
	return host
}

func extractHost(address string) string {
	addr, _, err := net.SplitHostPort(address)
	if err != nil {
		addr = address
	}
	return addr
}
