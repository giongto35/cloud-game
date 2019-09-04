package dtls

import (
	"net"
)

// Export extracts dtls state and inner connection from an already handshaked dtls conn
func (c *Conn) Export() (*State, net.Conn, error) {
	state, err := c.state.clone()
	if err != nil {
		return nil, nil, err
	}
	return state, c.nextConn, nil
}

// Resume imports an already stablished dtls connection using a specific dtls state
func Resume(state *State, conn net.Conn, config *Config) (*Conn, error) {
	// Custom flight handler that sets imported data and signals as handshaked
	flightHandler := func(c *Conn) (bool, error) {
		c.state = *state
		c.signalHandshakeComplete()
		return true, nil
	}

	// Empty handshake handler, since handshake was already done
	handshakeHandler := func(c *Conn) error {
		return nil
	}

	c, err := createConn(conn, flightHandler, handshakeHandler, config, state.isClient)
	if err != nil {
		return nil, err
	}

	return c, c.getConnErr()
}
