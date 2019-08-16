package ice

import (
	"net"
	"strings"
)

// CandidateHost is a candidate of type host
type CandidateHost struct {
	candidateBase

	network string
}

// NewCandidateHost creates a new host candidate
func NewCandidateHost(network string, address string, port int, component uint16) (*CandidateHost, error) {
	c := &CandidateHost{
		candidateBase: candidateBase{
			address:       address,
			candidateType: CandidateTypeHost,
			component:     component,
			port:          port,
		},
		network: network,
	}

	if !strings.HasSuffix(address, ".local") {
		ip := net.ParseIP(address)
		if ip == nil {
			return nil, ErrAddressParseFailed
		}

		if err := c.setIP(ip); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *CandidateHost) setIP(ip net.IP) error {
	networkType, err := determineNetworkType(c.network, ip)
	if err != nil {
		return err
	}

	c.candidateBase.networkType = networkType
	c.candidateBase.resolvedAddr = &net.UDPAddr{IP: ip, Port: c.port}
	return nil
}
