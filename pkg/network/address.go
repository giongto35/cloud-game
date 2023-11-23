package network

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

type Address string

func (a *Address) Port() (int, error) {
	if len(string(*a)) == 0 {
		return 0, errors.New("no address")
	}
	addr := replaceAllExceptLast(string(*a), ":", "_")
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	if val, err := strconv.Atoi(port); err == nil {
		return val, nil
	}
	return 0, errors.New("port is not a number")
}

func replaceAllExceptLast(s, c, x string) string {
	return strings.Replace(s, c, x, strings.Count(s, c)-1)
}
