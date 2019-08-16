package ice

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
	"github.com/pion/turnc"
)

const (
	stunGatherTimeout = time.Second * 5
)

func localInterfaces(networkTypes []NetworkType) (ips []net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	var IPv4Requested, IPv6Requested bool
	for _, typ := range networkTypes {
		if typ.IsIPv4() {
			IPv4Requested = true
		}

		if typ.IsIPv6() {
			IPv6Requested = true
		}
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return ips
		}

		for _, addr := range addrs {
			var ip net.IP
			switch addr := addr.(type) {
			case *net.IPNet:
				ip = addr.IP
			case *net.IPAddr:
				ip = addr.IP

			}
			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ipv4 := ip.To4(); ipv4 == nil {
				if !IPv6Requested {
					continue
				} else if !isSupportedIPv6(ip) {
					continue
				}
			} else if !IPv4Requested {
				continue
			}

			ips = append(ips, ip)
		}
	}
	return ips
}

func listenUDP(portMax, portMin int, network string, laddr *net.UDPAddr) (*net.UDPConn, error) {
	if (laddr.Port != 0) || ((portMin == 0) && (portMax == 0)) {
		return net.ListenUDP(network, laddr)
	}
	var i, j int
	i = portMin
	if i == 0 {
		i = 1
	}
	j = portMax
	if j == 0 {
		j = 0xFFFF
	}
	for i <= j {
		c, e := net.ListenUDP(network, &net.UDPAddr{IP: laddr.IP, Port: i})
		if e == nil {
			return c, e
		}
		i++
	}
	return nil, ErrPort
}

// GatherCandidates initiates the trickle based gathering process.
func (a *Agent) GatherCandidates() error {
	gatherErrChan := make(chan error, 1)

	runErr := a.run(func(agent *Agent) {
		if a.gatheringState != GatheringStateNew {
			gatherErrChan <- ErrMultipleGatherAttempted
			return
		} else if a.onCandidateHdlr == nil {
			gatherErrChan <- ErrNoOnCandidateHandler
			return
		}

		go a.gatherCandidates()

		gatherErrChan <- nil
	})
	if runErr != nil {
		return runErr
	}
	return <-gatherErrChan
}

func (a *Agent) gatherCandidates() {
	gatherStateUpdated := make(chan bool)
	if err := a.run(func(agent *Agent) {
		a.gatheringState = GatheringStateGathering
		close(gatherStateUpdated)
	}); err != nil {
		a.log.Warnf("failed to set gatheringState to GatheringStateGathering for gatherCandidates: %v", err)
		return
	}
	<-gatherStateUpdated

	for _, t := range a.candidateTypes {
		switch t {
		case CandidateTypeHost:
			a.gatherCandidatesLocal(a.networkTypes)
		case CandidateTypeServerReflexive:
			a.gatherCandidatesSrflx(a.urls, a.networkTypes)
		case CandidateTypeRelay:
			if err := a.gatherCandidatesRelay(a.urls); err != nil {
				a.log.Errorf("Failed to gather relay candidates: %v\n", err)
			}
		}
	}

	if err := a.run(func(agent *Agent) {
		if a.onCandidateHdlr != nil {
			go a.onCandidateHdlr(nil)
		}
	}); err != nil {
		a.log.Warnf("Failed to run onCandidateHdlr task: %v\n", err)
		return
	}

	if err := a.run(func(agent *Agent) {
		a.gatheringState = GatheringStateComplete
	}); err != nil {
		a.log.Warnf("Failed to update gatheringState: %v\n", err)
		return
	}
}

func (a *Agent) gatherCandidatesLocal(networkTypes []NetworkType) {
	var wg sync.WaitGroup
	defer wg.Wait()

	localIPs := localInterfaces(networkTypes)
	wg.Add(len(localIPs) * len(supportedNetworks))
	for _, ip := range localIPs {
		for _, network := range supportedNetworks {
			go func(network string, ip net.IP) {
				defer wg.Done()
				conn, err := listenUDP(int(a.portmax), int(a.portmin), network, &net.UDPAddr{IP: ip, Port: 0})
				if err != nil {
					a.log.Warnf("could not listen %s %s\n", network, ip)
					return
				}

				address := ip.String()
				if a.mDNSMode == MulticastDNSModeQueryAndGather {
					address = a.mDNSName
				}

				port := conn.LocalAddr().(*net.UDPAddr).Port
				c, err := NewCandidateHost(network, address, port, ComponentRTP)
				if err != nil {
					a.log.Warnf("Failed to create host candidate: %s %s %d: %v\n", network, ip, port, err)
					return
				}

				if a.mDNSMode == MulticastDNSModeQueryAndGather {
					if err = c.setIP(ip); err != nil {
						a.log.Warnf("Failed to create host candidate: %s %s %d: %v\n", network, ip, port, err)
						return
					}
				}

				if err := a.run(func(agent *Agent) {
					a.addCandidate(c)
				}); err != nil {
					a.log.Warnf("Failed to append to localCandidates: %v\n", err)
					return
				}

				c.start(a, conn)

				if err := a.run(func(agent *Agent) {
					if a.onCandidateHdlr != nil {
						go a.onCandidateHdlr(c)
					}
				}); err != nil {
					a.log.Warnf("Failed to run onCandidateHdlr task: %v\n", err)
					return
				}
			}(network, ip)
		}
	}
}

func (a *Agent) gatherCandidatesSrflx(urls []*URL, networkTypes []NetworkType) {
	for _, networkType := range networkTypes {
		network := networkType.String()
		for _, url := range urls {
			if url.Scheme != SchemeTypeSTUN {
				continue
			}

			hostPort := fmt.Sprintf("%s:%d", url.Host, url.Port)
			serverAddr, err := net.ResolveUDPAddr(network, hostPort)
			if err != nil {
				a.log.Warnf("failed to resolve stun host: %s: %v", hostPort, err)
				continue
			}

			conn, err := listenUDP(int(a.portmax), int(a.portmin), network, &net.UDPAddr{IP: nil, Port: 0})
			if err != nil {
				a.log.Warnf("Failed to listen on %s for %s: %v\n", conn.LocalAddr().String(), serverAddr.String(), err)
				continue
			}

			xoraddr, err := getXORMappedAddr(conn, serverAddr, stunGatherTimeout)
			if err != nil {
				a.log.Warnf("could not get server reflexive address %s %s: %v\n", network, url, err)
				continue
			}

			laddr := conn.LocalAddr().(*net.UDPAddr)
			ip := xoraddr.IP
			port := xoraddr.Port
			relIP := laddr.IP.String()
			relPort := laddr.Port
			c, err := NewCandidateServerReflexive(network, ip.String(), port, ComponentRTP, relIP, relPort)
			if err != nil {
				a.log.Warnf("Failed to create server reflexive candidate: %s %s %d: %v\n", network, ip, port, err)
				continue
			}

			if err := a.run(func(agent *Agent) {
				a.addCandidate(c)
			}); err != nil {
				a.log.Warnf("Failed to append to localCandidates: %v\n", err)
				continue
			}

			c.start(a, conn)

			if err := a.run(func(agent *Agent) {
				if a.onCandidateHdlr != nil {
					go a.onCandidateHdlr(c)
				}
			}); err != nil {
				a.log.Warnf("Failed to run onCandidateHdlr task: %v\n", err)
				continue
			}
		}
	}

}

func (a *Agent) gatherCandidatesRelay(urls []*URL) error {
	network := NetworkTypeUDP4.String() // TODO IPv6
	for _, url := range urls {
		switch {
		case url.Scheme != SchemeTypeTURN:
			continue
		case url.Username == "":
			return ErrUsernameEmpty
		case url.Password == "":
			return ErrPasswordEmpty
		}

		raddr, err := net.ResolveUDPAddr(network, fmt.Sprintf("%s:%d", url.Host, url.Port))
		if err != nil {
			return err
		}
		c, err := net.DialUDP(network, nil, raddr)
		if err != nil {
			return err
		}
		client, clientErr := turnc.New(turnc.Options{
			Conn:     c,
			Username: url.Username,
			Password: url.Password,
		})
		if clientErr != nil {
			return clientErr
		}
		allocation, allocErr := client.Allocate()
		if allocErr != nil {
			return allocErr
		}

		laddr := c.LocalAddr().(*net.UDPAddr)
		ip := allocation.Relayed().IP
		port := allocation.Relayed().Port

		candidate, err := NewCandidateRelay(network, ip.String(), port, ComponentRTP, laddr.IP.String(), laddr.Port)
		if err != nil {
			a.log.Warnf("Failed to create server reflexive candidate: %s %s %d: %v\n", network, ip, port, err)
			continue
		}
		candidate.setAllocation(client, allocation)

		a.addCandidate(candidate)
		candidate.start(a, nil)
	}

	return nil
}

// getXORMappedAddr initiates a stun requests to serverAddr using conn, reads the response and returns
// the XORMappedAddress returned by the stun server.
//
// Adapted from stun v0.2.
func getXORMappedAddr(conn *net.UDPConn, serverAddr net.Addr, deadline time.Duration) (*stun.XORMappedAddress, error) {
	if deadline > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(deadline)); err != nil {
			return nil, err
		}
	}
	defer func() {
		if deadline > 0 {
			_ = conn.SetReadDeadline(time.Time{})
		}
	}()
	resp, err := stunRequest(
		conn.Read,
		func(b []byte) (int, error) {
			return conn.WriteTo(b, serverAddr)
		},
	)
	if err != nil {
		return nil, err
	}
	var addr stun.XORMappedAddress
	if err = addr.GetFrom(resp); err != nil {
		return nil, fmt.Errorf("failed to get XOR-MAPPED-ADDRESS response: %v", err)
	}
	return &addr, nil
}

func stunRequest(read func([]byte) (int, error), write func([]byte) (int, error)) (*stun.Message, error) {
	req, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		return nil, err
	}
	if _, err = write(req.Raw); err != nil {
		return nil, err
	}
	const maxMessageSize = 1280
	bs := make([]byte, maxMessageSize)
	n, err := read(bs)
	if err != nil {
		return nil, err
	}
	res := &stun.Message{Raw: bs[:n]}
	if err := res.Decode(); err != nil {
		return nil, err
	}
	return res, nil
}
