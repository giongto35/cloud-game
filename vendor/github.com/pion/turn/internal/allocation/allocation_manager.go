package allocation

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/transport/vnet"
	"github.com/pkg/errors"
)

// ManagerConfig a bag of config params for Manager.
type ManagerConfig struct {
	LeveledLogger logging.LeveledLogger
	Net           *vnet.Net
}

// Manager is used to hold active allocations
type Manager struct {
	lock        sync.RWMutex
	allocations map[string]*Allocation
	log         logging.LeveledLogger
	net         *vnet.Net
}

// NewManager creates a new instance of Manager.
func NewManager(config *ManagerConfig) *Manager {
	if config.Net == nil {
		config.Net = vnet.NewNet(nil) // defaults to native operation
	}
	return &Manager{
		log:         config.LeveledLogger,
		net:         config.Net,
		allocations: make(map[string]*Allocation, 64),
	}
}

// GetAllocation fetches the allocation matching the passed FiveTuple
func (m *Manager) GetAllocation(fiveTuple *FiveTuple) *Allocation {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.allocations[fiveTuple.Fingerprint()]
}

// Close closes the manager and closes all allocations it manages
func (m *Manager) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, a := range m.allocations {
		if err := a.Close(); err != nil {
			return err
		}

	}
	return nil
}

// CreateAllocation creates a new allocation and starts relaying
func (m *Manager) CreateAllocation(
	fiveTuple *FiveTuple,
	turnSocket net.PacketConn,
	relayIP net.IP, // nolint:interfacer
	requestedPort int,
	lifetime time.Duration) (*Allocation, error) {

	if fiveTuple == nil {
		return nil, errors.Errorf("Allocations must not be created with nil FivTuple")
	}
	if fiveTuple.SrcAddr == nil {
		return nil, errors.Errorf("Allocations must not be created with nil FiveTuple.SrcAddr")
	}
	if fiveTuple.DstAddr == nil {
		return nil, errors.Errorf("Allocations must not be created with nil FiveTuple.DstAddr")
	}
	if a := m.GetAllocation(fiveTuple); a != nil {
		return nil, errors.Errorf("Allocation attempt created with duplicate FiveTuple %v", fiveTuple)
	}
	if turnSocket == nil {
		return nil, errors.Errorf("Allocations must not be created with nil turnSocket")
	}
	if lifetime == 0 {
		return nil, errors.Errorf("Allocations must not be created with a lifetime of 0")
	}

	a := NewAllocation(turnSocket, fiveTuple, m.log)

	network := "udp4"
	relayAddr := fmt.Sprintf("%s:%d", relayIP.String(), requestedPort)
	conn, err := m.net.ListenPacket(network, relayAddr)
	if err != nil {
		return nil, err
	}

	m.log.Debugf("listening on relay addr: %s", conn.LocalAddr().String())

	a.RelaySocket = conn
	a.RelayAddr = conn.LocalAddr()

	a.lifetimeTimer = time.AfterFunc(lifetime, func() {
		m.DeleteAllocation(a.fiveTuple)
	})

	m.lock.Lock()
	m.allocations[fiveTuple.Fingerprint()] = a
	m.lock.Unlock()

	go a.packetHandler(m)
	return a, nil
}

// DeleteAllocation removes an allocation
func (m *Manager) DeleteAllocation(fiveTuple *FiveTuple) {
	fingerprint := fiveTuple.Fingerprint()

	m.lock.Lock()
	allocation := m.allocations[fingerprint]
	delete(m.allocations, fingerprint)
	m.lock.Unlock()

	if allocation == nil {
		return
	}

	if err := allocation.Close(); err != nil {
		m.log.Errorf("Failed to close allocation: %v", err)
	}
}
