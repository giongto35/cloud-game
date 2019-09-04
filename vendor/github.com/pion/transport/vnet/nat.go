package vnet

import (
	"fmt"
	"sync"
	"time"

	"github.com/pion/logging"
)

// EndpointDependencyType defines a type of behavioral dependendency on the
// remote endpoint's IP address or port number. This is used for the two
// kinds of behaviors:
//  - Port mapping behavior
//  - Filtering behavior
// See: https://tools.ietf.org/html/rfc4787
type EndpointDependencyType uint8

const (
	// EndpointIndependent means the behavior is independent of the endpoint's address or port
	EndpointIndependent EndpointDependencyType = iota
	// EndpointAddrDependent means the behavior is dependent on the endpoint's address
	EndpointAddrDependent
	// EndpointAddrPortDependent means the behavior is dependent on the endpoint's address and port
	EndpointAddrPortDependent
)

const (
	defaultNATMappingLifeTime = 30 * time.Second
)

// NATType has a set of parameters that define the behavior of NAT.
type NATType struct {
	MappingBehavior   EndpointDependencyType
	FilteringBehavior EndpointDependencyType
	Hairpining        bool // Not implemented yet
	PortPreservation  bool // Not implemented yet
	MappingLifeTime   time.Duration
}

type natConfig struct {
	name          string
	natType       NATType
	mappedIP      string
	loggerFactory logging.LoggerFactory
}

type mapping struct {
	proto   string    // "udp" or "tcp"
	local   string    // "<ip-addr>:<port>"
	mapped  string    // "<ip-addr>:<port>"
	filter  string    // ":[<ip-addr>[:<port>]]"
	expires time.Time // time to expire
}

type networkAddressTranslator struct {
	name           string
	natType        NATType
	mappedIP       string
	outboundMap    map[string]*mapping // key: "<proto>:<local-ip>:<local-port>[:remote-ip[:remote-port]]
	inboundMap     map[string]*mapping // key: "<proto>:<mapped-ip>:<mapped-port>[:remote-ip[:remote-port]]"
	udpPortCounter int
	mutex          sync.RWMutex
	log            logging.LeveledLogger
}

func newNAT(config *natConfig) *networkAddressTranslator {
	natType := config.natType
	if natType.MappingLifeTime == 0 {
		natType.MappingLifeTime = defaultNATMappingLifeTime
	}

	return &networkAddressTranslator{
		name:        config.name,
		natType:     natType,
		mappedIP:    config.mappedIP,
		outboundMap: map[string]*mapping{},
		inboundMap:  map[string]*mapping{},
		log:         config.loggerFactory.NewLogger("vnet"),
	}

}

func (n *networkAddressTranslator) translateOutbound(from Chunk) (Chunk, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	to := from.Clone()

	if from.Network() == "udp" {
		var filter string
		switch n.natType.FilteringBehavior {
		case EndpointIndependent:
			filter = ""
		case EndpointAddrDependent:
			filter = fmt.Sprintf(":%s", from.getDestinationIP().String())
		case EndpointAddrPortDependent:
			filter = fmt.Sprintf(":%s", from.DestinationAddr().String())
		}

		oKey := fmt.Sprintf("udp:%s%s", from.SourceAddr().String(), filter)

		m := n.findOutboundMapping(oKey)
		if m == nil {
			// Create a new mapping
			mappedPort := 0xC000 + n.udpPortCounter
			n.udpPortCounter++

			m = &mapping{
				proto:   from.SourceAddr().Network(),
				local:   from.SourceAddr().String(),
				mapped:  fmt.Sprintf("%s:%d", n.mappedIP, mappedPort),
				filter:  filter,
				expires: time.Now().Add(n.natType.MappingLifeTime),
			}

			n.outboundMap[oKey] = m

			iKey := fmt.Sprintf("udp:%s:%d%s", n.mappedIP, mappedPort, filter)

			n.log.Debugf("[%s] created a new NAT binding oKey=%s iKey=%s\n",
				n.name,
				oKey,
				iKey)
			n.inboundMap[iKey] = m
		}

		if err := to.setSourceAddr(m.mapped); err != nil {
			return nil, err
		}

		n.log.Debugf("[%s] translate outbound chunk from %s to %s", n.name, from.String(), to.String())

		return to, nil
	}

	// TODO
	//if c.Network() == "tcp" {
	//}

	return nil, fmt.Errorf("non-udp translation is not supported yet")
}

func (n *networkAddressTranslator) translateInbound(from Chunk) (Chunk, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	to := from.Clone()

	if from.Network() == "udp" {
		var iKey string

		switch n.natType.FilteringBehavior {
		case EndpointIndependent:
			iKey = fmt.Sprintf("udp:%s",
				from.DestinationAddr().String(),
			)
		case EndpointAddrDependent:
			iKey = fmt.Sprintf("udp:%s:%s",
				from.DestinationAddr().String(),
				from.getSourceIP().String(),
			)
		case EndpointAddrPortDependent:
			iKey = fmt.Sprintf("udp:%s:%s",
				from.DestinationAddr().String(),
				from.SourceAddr().String(),
			)
		}

		m := n.findInboundMapping(iKey)
		if m == nil {
			return nil, fmt.Errorf("drop %s as no NAT binding found", from.String())
		}

		if err := to.setDestinationAddr(m.local); err != nil {
			return nil, err
		}

		n.log.Debugf("[%s] translate inbound chunk from %s to %s", n.name, from.String(), to.String())

		return to, nil
	}

	// TODO
	//if c.Network() == "tcp" {
	//}

	return nil, fmt.Errorf("non-udp translation is not supported yet")
}

// caller must hold the mutex
func (n *networkAddressTranslator) findOutboundMapping(oKey string) *mapping {
	now := time.Now()

	m, ok := n.outboundMap[oKey]
	if ok {
		// check if this mapping is expired
		if now.After(m.expires) {
			n.removeMapping(m)
			m = nil // expired
		} else {
			m.expires = time.Now().Add(n.natType.MappingLifeTime)
		}
	}

	return m
}

// caller must hold the mutex
func (n *networkAddressTranslator) findInboundMapping(iKey string) *mapping {
	now := time.Now()

	m, ok := n.inboundMap[iKey]
	if ok {
		// check if this mapping is expired
		if now.After(m.expires) {
			n.removeMapping(m)
			m = nil // expired
		}

		// See RFC 4847 Section 4.3.  Mapping Refresh
		// a) Inbound refresh may be useful for applications with no outgoing
		//   UDP traffic.  However, allowing inbound refresh may allow an
		//   external attacker or misbehaving application to keep a mapping
		//   alive indefinitely.  This may be a security risk.  Also, if the
		//   process is repeated with different ports, over time, it could
		//   use up all the ports on the NAT.
	}

	return m
}

// caller must hold the mutex
func (n *networkAddressTranslator) removeMapping(m *mapping) {
	oKey := fmt.Sprintf("%s:%s%s",
		m.proto,
		m.local,
		m.filter)

	iKey := fmt.Sprintf("%s:%s%s",
		m.proto,
		m.mapped,
		m.filter)

	delete(n.outboundMap, oKey)
	delete(n.inboundMap, iKey)
}
