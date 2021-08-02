package httpx

import (
	"net"
	"strconv"
	"strings"
)

type Address string

// SplitHostPort splits a network address of the form "host:port".
// Works similarly to net.SplitHostPort except it returns port value as a number.
func (a Address) SplitHostPort() (host string, port int) {
	address := string(a)
	// it's all an address without the colon
	if !strings.Contains(address, ":") {
		return address, 0
	}

	h, p, er := net.SplitHostPort(string(a))
	if er == nil {
		host = h
		if val, er := strconv.Atoi(p); er == nil {
			port = val
		}
	}
	return
}
