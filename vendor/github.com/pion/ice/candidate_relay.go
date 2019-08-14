package ice

import (
	"errors"
	"io"
	"net"

	"github.com/pion/turnc"
)

// CandidateRelay ...
type CandidateRelay struct {
	candidateBase

	allocation  *turnc.Allocation
	client      io.Closer
	permissions map[string]*turnc.Permission
}

// NewCandidateRelay creates a new relay candidate
func NewCandidateRelay(network string, address string, port int, component uint16, relAddr string, relPort int) (*CandidateRelay, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, ErrAddressParseFailed
	}

	networkType, err := determineNetworkType(network, ip)
	if err != nil {
		return nil, err
	}

	return &CandidateRelay{
		candidateBase: candidateBase{
			networkType:   networkType,
			candidateType: CandidateTypeRelay,
			address:       address,
			port:          port,
			resolvedAddr:  &net.UDPAddr{IP: ip, Port: port},
			component:     component,
			relatedAddress: &CandidateRelatedAddress{
				Address: relAddr,
				Port:    relPort,
			},
		},
		permissions: map[string]*turnc.Permission{},
	}, nil
}

func (c *CandidateRelay) setAllocation(client io.Closer, a *turnc.Allocation) {
	c.allocation = a
	c.client = client
}

func (c *CandidateRelay) start(a *Agent, conn net.PacketConn) {
	c.currAgent = a
}

func (c *CandidateRelay) close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, p := range c.permissions {
		if err := p.Close(); err != nil {
			return err
		}
	}
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *CandidateRelay) addPermission(dst Candidate) error {
	permission, err := c.allocation.Create(dst.addr())
	if err != nil {
		return err
	}

	c.lock.Lock()
	c.permissions[dst.String()] = permission
	if err = c.permissions[dst.String()].Bind(); err != nil {
		c.agent().log.Warnf("Failed to Create ChannelBind for %v: %v", dst.String, err)
	}
	c.lock.Unlock()

	go func(remoteAddr net.Addr) {
		log := c.agent().log
		buffer := make([]byte, receiveMTU)
		for {
			n, err := permission.Read(buffer)
			if err != nil {
				return
			}

			handleInboundCandidateMsg(c, buffer[:n], remoteAddr, log)
		}
	}(dst.addr())
	return nil
}

func (c *CandidateRelay) writeTo(raw []byte, dst Candidate) (int, error) {
	permission, ok := c.permissions[dst.String()]
	if !ok {
		return 0, errors.New("no permission created for remote candidate")
	}

	return permission.Write(raw)
}
