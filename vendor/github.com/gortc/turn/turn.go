// Package turn implements RFC 5766 Traversal Using Relays around NAT.
package turn

import (
	"encoding/binary"

	"github.com/pion/stun"
)

// bin is shorthand for binary.BigEndian.
var bin = binary.BigEndian

// Default ports for TURN from RFC 5766 Section 4.
const (
	// DefaultPort for TURN is same as STUN.
	DefaultPort = stun.DefaultPort
	// DefaultTLSPort is for TURN over TLS and is same as STUN.
	DefaultTLSPort = stun.DefaultTLSPort
)

var (
	// AllocateRequest is shorthand for allocation request message type.
	AllocateRequest = stun.NewType(stun.MethodAllocate, stun.ClassRequest)
	// CreatePermissionRequest is shorthand for create permission request type.
	CreatePermissionRequest = stun.NewType(stun.MethodCreatePermission, stun.ClassRequest)
	// SendIndication is shorthand for send indication message type.
	SendIndication = stun.NewType(stun.MethodSend, stun.ClassIndication)
	// RefreshRequest is shorthand for refresh request message type.
	RefreshRequest = stun.NewType(stun.MethodRefresh, stun.ClassRequest)
)
