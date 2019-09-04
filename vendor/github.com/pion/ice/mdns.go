package ice

import (
	"crypto/rand"
)

// MulticastDNSMode represents the different Multicast modes ICE can run in
type MulticastDNSMode byte

// MulticastDNSMode enum
const (
	// MulticastDNSModeDisabled means remote mDNS candidates will be discarded, and local host candidates will use IPs
	MulticastDNSModeDisabled MulticastDNSMode = iota + 1

	// MulticastDNSModeQueryOnly means remote mDNS candidates will be accepted, and local host candidates will use IPs
	MulticastDNSModeQueryOnly

	// MulticastDNSModeQueryAndGather means remote mDNS candidates will be accepted, and local host candidates will use mDNS
	MulticastDNSModeQueryAndGather
)

func generateMulticastDNSName() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b) //nolint

	if err != nil {
		return "", err
	}

	return generateRandString("", ".local")
}
