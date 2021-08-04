package httpx

import (
	"net"
	"strconv"
)

// mergeAddresses joins network host from the first param
// with the port value of a listener from the second param.
//
// As example, address host.com:8080 and listener 123.123.123.123:8888 will be
// transformed to host.com:8888.
func mergeAddresses(address string, l Listener) string {
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
	return addr
}
