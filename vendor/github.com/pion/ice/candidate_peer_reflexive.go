package ice

import "net"

// CandidatePeerReflexive ...
type CandidatePeerReflexive struct {
	candidateBase
}

// NewCandidatePeerReflexive creates a new peer reflective candidate
func NewCandidatePeerReflexive(network string, address string, port int, component uint16, relAddr string, relPort int) (*CandidatePeerReflexive, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, ErrAddressParseFailed
	}

	networkType, err := determineNetworkType(network, ip)
	if err != nil {
		return nil, err
	}

	return &CandidatePeerReflexive{
		candidateBase: candidateBase{
			networkType:   networkType,
			candidateType: CandidateTypePeerReflexive,
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
