package httpx

import (
	"errors"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

const maxPortRollAttempts = 42
const RollPorts = true

type Listener struct {
	net.Listener
}

func NewListener(address string, rollPorts bool) (*Listener, error) {
	ls, err := net.Listen("tcp4", address)
	if err != nil {
		if rollPorts && isErrorAddressAlreadyInUse(err) {
			host, port := Address(address).SplitHostPort()
			for i := port + 1; i < port+maxPortRollAttempts; i++ {
				log.Printf("ROLL %v %v", host, i)
				ls, err = net.Listen("tcp4", host+":"+strconv.Itoa(i))
				if err == nil {
					return &Listener{ls}, err
				}
				log.Printf("ERR -> %v", err)
			}
		}
		return nil, err
	}
	return &Listener{ls}, err
}

func isErrorAddressAlreadyInUse(err error) bool {
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
