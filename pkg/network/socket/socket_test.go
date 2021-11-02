package socket

import (
	"net"
	"testing"
)

func TestFailOnPortInUse(t *testing.T) {
	l, err := NewSocket("udp", 1234)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer l.(*net.UDPConn).Close()
	_, err = NewSocket("udp", 1234)
	if err == nil {
		t.Errorf("expected busy port error, but got none")
	}
}

func TestListenerPortRoll(t *testing.T) {
	l, err := NewSocketPortRoll("udp", 1234)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	defer l.(*net.UDPConn).Close()
	l2, err := NewSocketPortRoll("udp", 1234)
	if err != nil {
		t.Errorf("expected no port error, but got one")
	}
	l2.(*net.UDPConn).Close()
}
