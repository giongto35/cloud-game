package httpx

import (
	"errors"
	"net"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/network/socket"
)

const listenAttempts = 42

type Listener struct {
	net.Listener
}

func NewListener(address string, withNextFreePort bool) (*Listener, error) {
	listener, err := listener(address, withNextFreePort)
	if err != nil {
		return nil, err
	}
	return &Listener{listener}, err
}

func listener(address string, withNextFreePort bool) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err == nil || !withNextFreePort || !socket.IsPortBusyError(err) {
		return listener, err
	}
	// we will roll next available port
	host, prt, err := net.SplitHostPort(address)
	if err != nil {
		return listener, err
	}
	// it should be impossible to get 0 port here
	// or that's going to break otherwise
	port, err := strconv.Atoi(prt)
	if err != nil {
		return listener, err
	}
	for i := port + 1; i < port+listenAttempts; i++ {
		listener, err := net.Listen("tcp", host+":"+strconv.Itoa(i))
		if err == nil {
			return listener, nil
		}
	}
	return nil, errors.New("no available ports")
}

func (l Listener) GetPort() int {
	if l.Listener == nil {
		return 0
	}
	tcp, ok := l.Addr().(*net.TCPAddr)
	if ok && tcp != nil {
		return tcp.Port
	}
	return 0
}
