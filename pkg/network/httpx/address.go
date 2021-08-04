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
func mergeAddresses(address string, l net.Listener) string {
	addr, _, err := net.SplitHostPort(address)
	if err != nil {
		addr = address
	}
	if addr == "" {
		addr = "localhost"
	}

	if l != nil {
		tcp, ok := l.Addr().(*net.TCPAddr)
		if ok && tcp != nil && tcp.Port > 0 && tcp.Port != 80 && tcp.Port != 443 {
			addr += ":" + strconv.Itoa(tcp.Port)
		}
	}
	return addr
}
