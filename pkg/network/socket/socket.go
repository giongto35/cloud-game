package socket

import (
	"errors"
	"net"
	"os"
	"runtime"
	"syscall"
)

type ProtoType string

const (
	UDP  ProtoType = "udp"
	UDP4 ProtoType = "udp4"
	UDP6 ProtoType = "udp6"
	TCP  ProtoType = "tcp"
	TCP4 ProtoType = "tcp4"
	TCP6 ProtoType = "tcp6"
)

const listenAttempts = 42
const udpBufferSize = 16 * 1024 * 1024

func NewSocket(proto ProtoType, port int) (listener interface{}, err error) {
	if listener, err = socket(proto, port); err == nil {
		return listener, nil
	}
	if IsPortBusyError(err) {
		for i := port + 1; i < port+listenAttempts; i++ {
			listener, err := socket(proto, i)
			if err == nil {
				return listener, nil
			}
		}
		return nil, errors.New("no available ports")
	}
	return nil, err
}

func socket(proto ProtoType, port int) (interface{}, error) {
	switch proto {
	case UDP, UDP4, UDP6:
		if l, err := net.ListenUDP(string(proto), &net.UDPAddr{Port: port}); err == nil {
			_ = l.SetReadBuffer(udpBufferSize)
			_ = l.SetWriteBuffer(udpBufferSize)
			return l, nil
		} else {
			return nil, err
		}
	case TCP, TCP4, TCP6:
		if l, err := net.ListenTCP(string(proto), &net.TCPAddr{Port: port}); err == nil {
			return l, nil
		} else {
			return nil, err
		}
	}
	return nil, errors.New("socket error")
}

func IsPortBusyError(err error) bool {
	if err == nil {
		return false
	}
	var eOsSyscall *os.SyscallError
	if !errors.As(err, &eOsSyscall) {
		return false
	}
	var errErrno syscall.Errno
	if !errors.As(eOsSyscall, &errErrno) {
		return false
	}
	if errErrno == syscall.EADDRINUSE {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}
