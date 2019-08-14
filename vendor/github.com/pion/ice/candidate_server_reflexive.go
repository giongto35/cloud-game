package ice

import "net"

// CandidateServerReflexive ...
type CandidateServerReflexive struct {
	candidateBase
}

// NewCandidateServerReflexive creates a new server reflective candidate
func NewCandidateServerReflexive(network string, address string, port int, component uint16, relAddr string, relPort int) (*CandidateServerReflexive, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, ErrAddressParseFailed
	}

	networkType, err := determineNetworkType(network, ip)
	if err != nil {
		return nil, err
	}

	return &CandidateServerReflexive{
		candidateBase: candidateBase{
			networkType:   networkType,
			candidateType: CandidateTypeServerReflexive,
			address:       address,
			port:          port,
			resolvedAddr:  &net.UDPAddr{IP: ip, Port: port},
			component:     component,
			relatedAddress: &CandidateRelatedAddress{
				Address: relAddr,
				Port:    relPort,
			},
		},
	}, nil
}
