package allocation

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type reservation struct {
	token string
	port  int
}

// ReservationManager is used to manage reservations
type ReservationManager struct {
	lock         sync.RWMutex
	reservations []*reservation
}

// CreateReservation stores the reservation for the token+port
func (m *ReservationManager) CreateReservation(reservationToken string, port int) {
	time.AfterFunc(30*time.Second, func() {
		m.lock.Lock()
		defer m.lock.Unlock()
		for i := len(m.reservations) - 1; i >= 0; i-- {
			if m.reservations[i].token == reservationToken {
				m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)
				return
			}
		}
	})

	m.lock.Lock()
	m.reservations = append(m.reservations, &reservation{
		token: reservationToken,
		port:  port,
	})
	m.lock.Unlock()
}

// GetReservation returns the port for a given reservation if it exists
func (m *ReservationManager) GetReservation(reservationToken string) (int, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, r := range m.reservations {
		if r.token == reservationToken {
			return r.port, true
		}
	}
	return 0, false
}

// GetRandomEvenPort returns a random un-allocated udp4 port
func GetRandomEvenPort() (int, error) {
	listener, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return 0, err
	}

	addr, ok := listener.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to cast net.Addr to *net.UDPAddr")
	} else if err := listener.Close(); err != nil {
		return 0, err
	} else if addr.Port%2 == 1 {
		return GetRandomEvenPort()
	}

	return addr.Port, nil
}
