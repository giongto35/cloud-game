package dtls

import (
	"errors"
	"net"

	"github.com/pion/dtls/internal/udp"
)

// Listen creates a DTLS listener
func Listen(network string, laddr *net.UDPAddr, config *Config) (*Listener, error) {
	if config == nil {
		return nil, errors.New("no config provided")
	}
	parent, err := udp.Listen(network, laddr)
	if err != nil {
		return nil, err
	}
	return &Listener{
		config: config,
		parent: parent,
	}, nil
}

// Listener represents a DTLS listener
type Listener struct {
	config *Config
	parent *udp.Listener
}

// Accept waits for and returns the next connection to the listener.
// You have to either close or read on all connection that are created.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.parent.Accept()
	if err != nil {
		return nil, err
	}
	return Server(c, l.config)
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
// Already Accepted connections are not closed.
func (l *Listener) Close() error {
	return l.parent.Close()
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.parent.Addr()
}
