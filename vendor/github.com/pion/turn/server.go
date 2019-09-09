package turn

import (
	"crypto/md5" // #nosec
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun"
	"github.com/pion/transport/vnet"
	"github.com/pion/turn/internal/allocation"
	"github.com/pion/turn/internal/proto"
	"github.com/pkg/errors"
)

const (
	maxStunMessageSize = 1500
)

// AuthHandler is a callback used to handle incoming auth requests, allowing users to customize Pion TURN
// with custom behavior
type AuthHandler func(username string, srcAddr net.Addr) (password string, ok bool)

// ServerConfig is a bag of config parameters for Server.
type ServerConfig struct {
	// Realm sets the realm for this server
	Realm string
	// AuthHandler is the handler called on each incoming auth requests.
	AuthHandler AuthHandler
	// ChannelBindTimeout sets the lifetime of channel binding. Defaults to 10 minutes.
	ChannelBindTimeout time.Duration
	// ListeningPort sets the listening port number. Defaults to 3478.
	ListeningPort int
	// LoggerFactory must be set for logging from this server.
	LoggerFactory logging.LoggerFactory
	// Net is used by pion developers. Do not use in your application.
	Net *vnet.Net
	// Software is the STUN SOFTWARE attribute. Useful for debugging purpose.
	Software string
	// Sender is a custom implementation of the request Sender.
	Sender Sender
}

type listener struct {
	conn    net.PacketConn
	closeCh chan struct{}
}

// Server is an instance of the Pion TURN server
type Server struct {
	lock               sync.RWMutex
	listeners          []*listener
	listenIPs          []net.IP
	relayIPs           []net.IP
	listenPort         int
	realm              string
	authHandler        AuthHandler
	manager            *allocation.Manager
	reservationManager *allocation.ReservationManager
	channelBindTimeout time.Duration
	log                logging.LeveledLogger
	net                *vnet.Net
	software           stun.Software
	sender             Sender
}

// NewServer creates the Pion TURN server
func NewServer(config *ServerConfig) *Server {
	log := config.LoggerFactory.NewLogger("turn")

	if config.Net == nil {
		config.Net = vnet.NewNet(nil) // defaults to native operation
	} else {
		log.Warn("vnet is enabled")
	}

	if config.Sender == nil {
		config.Sender = defaultBuildAndSend
	}

	manager := allocation.NewManager(&allocation.ManagerConfig{
		LeveledLogger: log,
		Net:           config.Net,
	})

	listenPort := config.ListeningPort
	if listenPort == 0 {
		listenPort = 3478
	}

	channelBindTimeout := config.ChannelBindTimeout
	if channelBindTimeout == 0 {
		channelBindTimeout = proto.DefaultLifetime
	}

	return &Server{
		listenPort:         listenPort,
		realm:              config.Realm,
		authHandler:        config.AuthHandler,
		manager:            manager,
		reservationManager: &allocation.ReservationManager{},
		channelBindTimeout: channelBindTimeout,
		log:                log,
		net:                config.Net,
		software:           stun.NewSoftware(config.Software),
		sender:             config.Sender,
	}
}

// AddListeningIPAddr adds a listening IP address.
// If not specified, it will automatically assigns the listening IP addresses
// from the system.
// This method must be called before calling Start().
func (s *Server) AddListeningIPAddr(addrStr string) error {
	ip := net.ParseIP(addrStr)
	if ip.To4() == nil {
		return fmt.Errorf("Non-IPv4 address is not supported")
	}

	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("link-local unicast address is not allowed")
	}
	s.listenIPs = append(s.listenIPs, ip)
	return nil
}

// AddRelayIPAddr adds a listening IP address.
// Note: current implementation can have only one relay IP address.
// If not specified, it will automatically assigns the relay IP addresses
// from the system.
// This method must be called before calling Start().
func (s *Server) AddRelayIPAddr(addrStr string) error {
	ip := net.ParseIP(addrStr)
	if ip.To4() == nil {
		return fmt.Errorf("Non-IPv4 address is not supported")
	}

	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("link-local unicast address is not allowed")
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified IP is not allowed")
	}
	s.relayIPs = append(s.relayIPs, ip)
	return nil
}

// caller must hold the mutex
func (s *Server) gatherSystemIPAddrs() ([]net.IP, error) {
	s.log.Debug("gathering local IP address...")

	var ips []net.IP

	ifs, err := s.net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 {
			continue // skip if interface is not up
		}

		if ifc.Flags&net.FlagLoopback != 0 {
			continue // skip loopback address
		}

		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch addr := addr.(type) {
			case *net.IPNet:
				ip = addr.IP
			case *net.IPAddr:
				ip = addr.IP
			}

			if ip == nil {
				return nil, fmt.Errorf("invalid IP address: %s", addr.String())
			}

			if ip.To4() == nil {
				continue // skip non-IPv4 address
			}

			if ip.IsLinkLocalUnicast() {
				continue
			}

			s.log.Debugf("- found local IP: %s", ip.String())
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// Listen starts listening and handling TURN traffic
// caller must hold the mutex
func (s *Server) listen(localIP net.IP) error {
	network := "udp4"
	listenAddr := fmt.Sprintf("%s:%d", localIP.String(), s.listenPort)
	conn, err := s.net.ListenPacket(network, listenAddr)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to listen on %s", listenAddr))
	}

	laddr := conn.LocalAddr()
	s.log.Infof("Listening on %s:%s", laddr.Network(), laddr.String())

	closeCh := make(chan struct{})
	s.listeners = append(s.listeners, &listener{
		conn:    conn,
		closeCh: closeCh,
	})

	go func() {
		buf := make([]byte, maxStunMessageSize)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				s.log.Debugf("exit read loop on error: %s", err.Error())
				break
			}

			if err := s.handleUDPPacket(conn, addr, buf[:n]); err != nil {
				s.log.Error(err.Error())
			}
		}

		close(closeCh)
	}()

	return nil
}

// Start starts the server.
func (s *Server) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var ips []net.IP

	if len(s.listenIPs) == 0 {
		var err error
		ips, err = s.gatherSystemIPAddrs()
		if err != nil {
			return err
		}
		if len(ips) == 0 {
			return fmt.Errorf("no local IP address found")
		}

		s.listenIPs = ips
	}

	// If s.relayIPs is empty, use s.listenIPs
	if len(s.relayIPs) == 0 {
		s.relayIPs = s.listenIPs
	}

	for _, localIP := range s.listenIPs {
		err := s.listen(localIP)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the connection.
func (s *Server) Close() error {
	var toJoin []*listener
	var err error

	func() {
		s.lock.RLock()
		defer s.lock.RUnlock()

		if err2 := s.manager.Close(); err2 != nil {
			err = err2
		}

		for _, l := range s.listeners {
			err2 := l.conn.Close()
			if err2 != nil {
				s.log.Debugf("Close() returned error: %s", err2.Error())
				continue
			}
			toJoin = append(toJoin, l)
		}
	}()

	for _, l := range s.listeners {
		<-l.closeCh
	}

	return err
}

// caller must hold the mutex
func (s *Server) handleUDPPacket(conn net.PacketConn, srcAddr net.Addr, buf []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.log.Debugf("received %d bytes of udp from %s on %s",
		len(buf),
		srcAddr.String(),
		conn.LocalAddr().String(),
	)
	if proto.IsChannelData(buf) {
		return s.handleDataPacket(conn, srcAddr, buf)
	}

	return s.handleTURNPacket(conn, srcAddr, buf)
}

// caller must hold the mutex
func (s *Server) handleDataPacket(conn net.PacketConn, srcAddr net.Addr, buf []byte) error {
	s.log.Debugf("received DataPacket from %s", srcAddr.String())
	c := proto.ChannelData{Raw: buf}
	if err := c.Decode(); err != nil {
		return errors.Wrap(err, "Failed to create channel data from packet")
	}

	err := s.handleChannelData(conn, srcAddr, &c)
	if err != nil {
		err = errors.Errorf("unable to handle ChannelData from %v: %v", srcAddr, err)
	}

	return err
}

// caller must hold the mutex
func (s *Server) handleTURNPacket(conn net.PacketConn, srcAddr net.Addr, buf []byte) error {
	s.log.Debug("handleTURNPacket")
	m := &stun.Message{Raw: append([]byte{}, buf...)}
	if err := m.Decode(); err != nil {
		return errors.Wrap(err, "failed to create stun message from packet")
	}

	h, err := s.getMessageHandler(m.Type.Class, m.Type.Method)
	if err != nil {
		return errors.Errorf("unhandled STUN packet %v-%v from %v: %v", m.Type.Method, m.Type.Class, srcAddr, err)
	}

	err = h(conn, srcAddr, m)
	if err != nil {
		return errors.Errorf("failed to handle %v-%v from %v: %v", m.Type.Method, m.Type.Class, srcAddr, err)
	}

	return nil
}

type messageHandler func(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error

func (s *Server) getMessageHandler(class stun.MessageClass, method stun.Method) (messageHandler, error) {
	switch class {
	case stun.ClassIndication:
		switch method {
		case stun.MethodSend:
			return s.handleSendIndication, nil
		default:
			return nil, errors.Errorf("unexpected method: %s", method)
		}

	case stun.ClassRequest:
		switch method {
		case stun.MethodAllocate:
			return s.handleAllocateRequest, nil
		case stun.MethodRefresh:
			return s.handleRefreshRequest, nil
		case stun.MethodCreatePermission:
			return s.handleCreatePermissionRequest, nil
		case stun.MethodChannelBind:
			return s.handleChannelBindRequest, nil
		case stun.MethodBinding:
			return s.handleBindingRequest, nil
		default:
			return nil, errors.Errorf("unexpected method: %s", method)
		}

	default:
		return nil, errors.Errorf("unexpected class: %s", class)
	}
}

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// TODO, include time info support stale nonces
func buildNonce() (string, error) {
	/* #nosec */
	h := md5.New()
	now := time.Now().Unix()
	if _, err := io.WriteString(h, strconv.FormatInt(now, 10)); err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Failed generating nonce %v \n", err))
	}
	if _, err := io.WriteString(h, strconv.FormatInt(rand.Int63(), 10)); err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Failed generating nonce %v \n", err))
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func assertMessageIntegrity(m *stun.Message, ourKey []byte) error {
	messageIntegrityAttr := stun.MessageIntegrity(ourKey)
	return messageIntegrityAttr.Check(m)
}
