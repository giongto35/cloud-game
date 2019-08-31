package turn

import (
	"net"

	"github.com/pion/stun"
)

// Sender is responsible for building a message and sending it to the given addr.
type Sender func(conn net.PacketConn, addr net.Addr, attrs ...stun.Setter) error

// defaultBuildAndSend is a sender default implementation.
func defaultBuildAndSend(conn net.PacketConn, dst net.Addr, attrs ...stun.Setter) error {
	msg, err := stun.Build(attrs...)
	if err != nil {
		return err
	}
	_, err = conn.WriteTo(msg.Raw, dst)
	return err
}
