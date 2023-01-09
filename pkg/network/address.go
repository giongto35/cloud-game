package network

import (
	"errors"
	"strconv"
	"strings"
)

type Address string

func (a *Address) Port() (int, error) {
	if len(string(*a)) == 0 {
		return 0, errors.New("no address")
	}
	parts := strings.Split(string(*a), ":")
	var port string
	if len(parts) == 1 {
		port = parts[0]
	} else {
		port = parts[len(parts)-1]
	}
	if val, err := strconv.Atoi(port); err == nil {
		return val, nil
	}
	return 0, errors.New("port is not a number")
}
