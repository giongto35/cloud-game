package network

import (
	"errors"
	"strconv"
	"strings"
)

type Address string

func (a *Address) Port() (error, int) {
	if len(string(*a)) == 0 {
		return errors.New("no address"), 0
	}
	parts := strings.Split(string(*a), ":")
	var port string
	if len(parts) == 1 {
		port = parts[0]
	} else {
		port = parts[len(parts)-1]
	}
	if val, err := strconv.Atoi(port); err == nil {
		return nil, val
	}
	return errors.New("port is not a number"), 0
}
