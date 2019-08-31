package turn

import (
	"crypto/md5" // #nosec
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/stun"
	"github.com/pion/turn/internal/allocation"
	"github.com/pion/turn/internal/ipnet"
	"github.com/pion/turn/internal/proto"
	"github.com/pkg/errors"
)

const (
	maximumLifetime = time.Hour // https://tools.ietf.org/html/rfc5766#section-6.2 defines 3600 seconds recommendation
)

type curriedSend func(class stun.MessageClass, method stun.Method, transactionID [stun.TransactionIDSize]byte, attrs ...stun.Setter) error

func authenticateRequest(curriedSend curriedSend, m *stun.Message, callingMethod stun.Method, realm string, authHandler AuthHandler, srcAddr net.Addr) (stun.MessageIntegrity, string, error) {
	handleErr := func(err error) (stun.MessageIntegrity, string, error) {
		if sendErr := curriedSend(stun.ClassErrorResponse, callingMethod, m.TransactionID,
			&stun.ErrorCodeAttribute{Code: stun.CodeBadRequest},
		); sendErr != nil {
			err = errors.Errorf(strings.Join([]string{sendErr.Error(), err.Error()}, "\n"))
		}
		return stun.MessageIntegrity{}, "", err
	}

	if !m.Contains(stun.AttrMessageIntegrity) {
		nonce, err := buildNonce()
		if err != nil {
			return stun.MessageIntegrity{}, "", err
		}

		return nil, "", curriedSend(stun.ClassErrorResponse, callingMethod, m.TransactionID,
			&stun.ErrorCodeAttribute{Code: stun.CodeUnauthorized},
			stun.NewNonce(nonce),
			stun.NewRealm(realm),
		)

	}

	var ourKey [16]byte
	nonceAttr := &stun.Nonce{}
	usernameAttr := &stun.Username{}
	realmAttr := &stun.Realm{}

	if err := realmAttr.GetFrom(m); err != nil {
		return handleErr(err)
	}

	if err := nonceAttr.GetFrom(m); err != nil {
		return handleErr(err)
	}

	if err := usernameAttr.GetFrom(m); err != nil {
		return handleErr(err)
	}

	password, ok := authHandler(usernameAttr.String(), srcAddr)
	if !ok {
		return handleErr(errors.Errorf("No user exists for %s", usernameAttr.String()))
	}

	/* #nosec */
	ourKey = md5.Sum([]byte(usernameAttr.String() + ":" + realmAttr.String() + ":" + password))
	if err := assertMessageIntegrity(m, ourKey[:]); err != nil {
		return handleErr(err)
	}

	return stun.NewLongTermIntegrity(usernameAttr.String(), realmAttr.String(), password), usernameAttr.String(), nil
}

func assertDontFragment(curriedSend curriedSend, m *stun.Message, attr stun.Setter) error {
	if m.Contains(stun.AttrDontFragment) {
		err := errors.Errorf("no support for DONT-FRAGMENT")
		if sendErr := curriedSend(stun.ClassErrorResponse, stun.MethodAllocate, m.TransactionID,
			&stun.ErrorCodeAttribute{Code: stun.CodeUnknownAttribute},
			&stun.UnknownAttributes{stun.AttrDontFragment},
			attr,
		); sendErr != nil {
			err = errors.Errorf(strings.Join([]string{sendErr.Error(), err.Error()}, "\n"))
		}
		return err
	}
	return nil
}

// https://tools.ietf.org/html/rfc5766#section-6.2
// caller must hold the mutex
func (s *Server) handleAllocateRequest(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error {
	s.log.Debugf("received AllocateRequest from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	curriedSend := func(class stun.MessageClass, method stun.Method, transactionID [stun.TransactionIDSize]byte, attrs ...stun.Setter) error {
		return s.sender(conn, srcAddr, s.makeAttrs(transactionID, stun.NewType(method, class), attrs...)...)
	}
	respondWithError := func(err error, messageIntegrity stun.MessageIntegrity, errorCode stun.ErrorCode) error {
		if sendErr := curriedSend(stun.ClassErrorResponse, stun.MethodAllocate, m.TransactionID,
			&stun.ErrorCodeAttribute{Code: errorCode},
			messageIntegrity,
		); sendErr != nil {
			err = errors.Errorf(strings.Join([]string{sendErr.Error(), err.Error()}, "\n"))
		}
		return err
	}

	// 1. The server MUST require that the request be authenticated.  This
	//    authentication MUST be done using the long-term credential
	//    mechanism of [https://tools.ietf.org/html/rfc5389#section-10.2.2]
	//    unless the client and server agree to use another mechanism through
	//    some procedure outside the scope of this document.
	messageIntegrity, _, err := authenticateRequest(curriedSend, m, stun.MethodAllocate, s.realm, s.authHandler, srcAddr)
	if err != nil {
		return err
	}
	if messageIntegrity == nil {
		return nil
	}

	fiveTuple := &allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	}
	requestedPort := 0
	reservationToken := ""

	// 2. The server checks if the 5-tuple is currently in use by an
	//    existing allocation.  If yes, the server rejects the request with
	//    a 437 (Allocation Mismatch) error.
	if alloc := s.manager.GetAllocation(fiveTuple); alloc != nil {
		return respondWithError(errors.Errorf("Relay already allocated for 5-TUPLE"), messageIntegrity, stun.CodeAllocMismatch)
	}

	// 3. The server checks if the request contains a REQUESTED-TRANSPORT
	//    attribute.  If the REQUESTED-TRANSPORT attribute is not included
	//    or is malformed, the server rejects the request with a 400 (Bad
	//    Request) error.  Otherwise, if the attribute is included but
	//    specifies a protocol other that UDP, the server rejects the
	//    request with a 442 (Unsupported Transport Protocol) error.
	var requestedTransport proto.RequestedTransport
	if err = requestedTransport.GetFrom(m); err != nil {
		return respondWithError(err, messageIntegrity, stun.CodeBadRequest)
	}
	if requestedTransport.Protocol != proto.ProtoUDP {
		return respondWithError(err, messageIntegrity, stun.CodeUnsupportedTransProto)
	}

	// 4. The request may contain a DONT-FRAGMENT attribute.  If it does,
	//    but the server does not support sending UDP datagrams with the DF
	//    bit set to 1 (see Section 12), then the server treats the DONT-
	//    FRAGMENT attribute in the Allocate request as an unknown
	//    comprehension-required attribute.
	if err = assertDontFragment(curriedSend, m, messageIntegrity); err != nil {
		return err
	}

	// 5.  The server checks if the request contains a RESERVATION-TOKEN
	//     attribute.  If yes, and the request also contains an EVEN-PORT
	//     attribute, then the server rejects the request with a 400 (Bad
	//     Request) error.  Otherwise, it checks to see if the token is
	//     valid (i.e., the token is in range and has not expired and the
	//     corresponding relayed transport address is still available).  If
	//     the token is not valid for some reason, the server rejects the
	//     request with a 508 (Insufficient Capacity) error.
	var reservationTokenAttr proto.ReservationToken
	if err = reservationTokenAttr.GetFrom(m); err == nil {
		var evenPort proto.EvenPort
		if err = evenPort.GetFrom(m); err == nil {
			return respondWithError(errors.Errorf("Request must not contain RESERVATION-TOKEN and EVEN-PORT"), messageIntegrity, stun.CodeBadRequest)
		}

		allocationPort, reservationFound := s.reservationManager.GetReservation(string(reservationTokenAttr))
		if !reservationFound {
			return respondWithError(errors.Errorf("No reservation found with token %s", string(reservationTokenAttr)), messageIntegrity, stun.CodeBadRequest)
		}
		requestedPort = allocationPort + 1
	}

	// 6. The server checks if the request contains an EVEN-PORT attribute.
	//    If yes, then the server checks that it can satisfy the request
	//    (i.e., can allocate a relayed transport address as described
	//    below).  If the server cannot satisfy the request, then the
	//    server rejects the request with a 508 (Insufficient Capacity)
	//    error.
	var evenPort proto.EvenPort
	if err = evenPort.GetFrom(m); err == nil {
		randomPort := 0
		randomPort, err = allocation.GetRandomEvenPort()
		if err != nil {
			return respondWithError(err, messageIntegrity, stun.CodeInsufficientCapacity)
		}
		requestedPort = randomPort
		reservationToken = randSeq(8)
	}

	// 7. At any point, the server MAY choose to reject the request with a
	//    486 (Allocation Quota Reached) error if it feels the client is
	//    trying to exceed some locally defined allocation quota.  The
	//    server is free to define this allocation quota any way it wishes,
	//    but SHOULD define it based on the username used to authenticate
	//    the request, and not on the client's transport address.

	// 8. Also at any point, the server MAY choose to reject the request
	//    with a 300 (Try Alternate) error if it wishes to redirect the
	//    client to a different server.  The use of this error code and
	//    attribute follow the specification in [RFC5389].
	// Check current usage vs redis usage of other servers
	// if bad, redirect { stun.AttrErrorCode, 300 }
	lifetimeDuration := allocationLifeTime(m)
	a, err := s.manager.CreateAllocation(
		fiveTuple, conn,
		s.relayIPs[0], // TODO: allow more than one relay IP
		requestedPort,
		lifetimeDuration)
	if err != nil {
		return respondWithError(err, messageIntegrity, stun.CodeInsufficientCapacity)
	}

	// Once the allocation is created, the server replies with a success
	// response.  The success response contains:
	//   * An XOR-RELAYED-ADDRESS attribute containing the relayed transport
	//     address.
	//   * A LIFETIME attribute containing the current value of the time-to-
	//     expiry timer.
	//   * A RESERVATION-TOKEN attribute (if a second relayed transport
	//     address was reserved).
	//   * An XOR-MAPPED-ADDRESS attribute containing the client's IP address
	//     and port (from the 5-tuple).

	srcIP, srcPort, err := ipnet.AddrIPPort(srcAddr)
	if err != nil {
		return respondWithError(err, messageIntegrity, stun.CodeBadRequest)
	}

	_, relayPort, err := ipnet.AddrIPPort(a.RelayAddr)
	if err != nil {
		return respondWithError(err, messageIntegrity, stun.CodeBadRequest)
	}

	dstIP, _, err := ipnet.AddrIPPort(dstAddr)
	if err != nil {
		return respondWithError(err, messageIntegrity, stun.CodeBadRequest)
	}

	responseAttrs := []stun.Setter{
		&proto.RelayedAddress{
			IP:   dstIP,
			Port: relayPort,
		},
		&proto.Lifetime{
			Duration: lifetimeDuration,
		},
		&stun.XORMappedAddress{
			IP:   srcIP,
			Port: srcPort,
		},
	}

	if reservationToken != "" {
		s.reservationManager.CreateReservation(reservationToken, relayPort)
		responseAttrs = append(responseAttrs, proto.ReservationToken([]byte(reservationToken)))
	}

	return curriedSend(stun.ClassSuccessResponse, stun.MethodAllocate, m.TransactionID, append(responseAttrs, messageIntegrity)...)
}

// caller must hold the mutex
func (s *Server) handleRefreshRequest(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error {
	s.log.Debugf("received RefreshRequest from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	curriedSend := func(class stun.MessageClass, method stun.Method, transactionID [stun.TransactionIDSize]byte, attrs ...stun.Setter) error {
		return s.sender(conn, srcAddr, s.makeAttrs(transactionID, stun.NewType(method, class), attrs...)...)
	}
	messageIntegrity, _, err := authenticateRequest(curriedSend, m, stun.MethodCreatePermission, s.realm, s.authHandler, srcAddr)
	if err != nil {
		return err
	}

	a := s.manager.GetAllocation(&allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	})
	if a == nil {
		return errors.Errorf("No allocation found for %v:%v", srcAddr, dstAddr)
	}

	lifetimeDuration := allocationLifeTime(m)
	a.Refresh(lifetimeDuration)

	return curriedSend(stun.ClassSuccessResponse, stun.MethodRefresh, m.TransactionID,
		&proto.Lifetime{
			Duration: lifetimeDuration,
		},
		messageIntegrity,
	)
}

// caller must hold the mutex
func (s *Server) handleCreatePermissionRequest(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error {
	s.log.Debugf("received CreatePermission from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	curriedSend := func(class stun.MessageClass, method stun.Method, transactionID [stun.TransactionIDSize]byte, attrs ...stun.Setter) error {
		return s.sender(conn, srcAddr, s.makeAttrs(transactionID, stun.NewType(method, class), attrs...)...)
	}

	a := s.manager.GetAllocation(&allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	})
	if a == nil {
		return errors.Errorf("No allocation found for %v:%v", srcAddr, dstAddr)
	}

	messageIntegrity, _, err := authenticateRequest(curriedSend, m, stun.MethodCreatePermission, s.realm, s.authHandler, srcAddr)
	if err != nil {
		return err
	}
	addCount := 0

	if err := m.ForEach(stun.AttrXORPeerAddress, func(m *stun.Message) error {
		var peerAddress proto.PeerAddress
		if err := peerAddress.GetFrom(m); err != nil {
			return err
		}

		s.log.Debugf("adding permission for %s", fmt.Sprintf("%s:%d",
			peerAddress.IP.String(), peerAddress.Port))
		a.AddPermission(allocation.NewPermission(
			&net.UDPAddr{
				IP:   peerAddress.IP,
				Port: peerAddress.Port,
			},
			s.log,
		))
		addCount++
		return nil
	}); err != nil {
		addCount = 0
	}

	respClass := stun.ClassSuccessResponse
	if addCount == 0 {
		respClass = stun.ClassErrorResponse
	}

	return curriedSend(respClass, stun.MethodCreatePermission, m.TransactionID,
		messageIntegrity)
}

// caller must hold the mutex
func (s *Server) handleSendIndication(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error {
	s.log.Debugf("received SendIndication from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	a := s.manager.GetAllocation(&allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	})
	if a == nil {
		return errors.Errorf("No allocation found for %v:%v", srcAddr, dstAddr)
	}

	dataAttr := proto.Data{}
	if err := dataAttr.GetFrom(m); err != nil {
		return err
	}

	peerAddress := proto.PeerAddress{}
	if err := peerAddress.GetFrom(m); err != nil {
		return err
	}

	msgDst := &net.UDPAddr{IP: peerAddress.IP, Port: peerAddress.Port}
	if perm := a.GetPermission(msgDst); perm == nil {
		return errors.Errorf("Unable to handle send-indication, no permission added: %v", msgDst)
	}

	l, err := a.RelaySocket.WriteTo(dataAttr, msgDst)
	if l != len(dataAttr) {
		return errors.Errorf("packet write smaller than packet %d != %d (expected) err: %v", l, len(dataAttr), err)
	}
	return err
}

// caller must hold the mutex
func (s *Server) handleChannelBindRequest(conn net.PacketConn, srcAddr net.Addr, m *stun.Message) error {
	s.log.Debugf("received ChannelBindRequest from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	errorSend := func(err error, attrs ...stun.Setter) error {
		sendErr := s.sender(conn, srcAddr, s.makeAttrs(m.TransactionID, stun.NewType(stun.MethodChannelBind, stun.ClassErrorResponse), attrs...)...)
		if sendErr != nil {
			err = errors.Errorf(strings.Join([]string{sendErr.Error(), err.Error()}, "\n"))
		}
		return err
	}

	a := s.manager.GetAllocation(&allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	})
	if a == nil {
		return errors.Errorf("No allocation found for %v:%v", srcAddr, dstAddr)
	}

	messageIntegrity, _, err := authenticateRequest(func(class stun.MessageClass, method stun.Method, transactionID [stun.TransactionIDSize]byte, attrs ...stun.Setter) error {
		return s.sender(conn, srcAddr, s.makeAttrs(m.TransactionID, stun.NewType(method, class))...)
	}, m, stun.MethodChannelBind, s.realm, s.authHandler, srcAddr)
	if err != nil {
		return errorSend(err, stun.CodeBadRequest)
	}

	var channel proto.ChannelNumber
	if err = channel.GetFrom(m); err != nil {
		return errorSend(err, stun.CodeBadRequest)
	}

	peerAddr := proto.PeerAddress{}
	if err = peerAddr.GetFrom(m); err != nil {
		return errorSend(err, stun.CodeBadRequest)
	}

	s.log.Debugf("binding channel %d to %s",
		channel,
		fmt.Sprintf("%s:%d", peerAddr.IP.String(), peerAddr.Port))
	err = a.AddChannelBind(allocation.NewChannelBind(
		channel,
		&net.UDPAddr{IP: peerAddr.IP, Port: peerAddr.Port},
		s.log,
	), s.channelBindTimeout)
	if err != nil {
		return errorSend(err, stun.CodeBadRequest)
	}

	return s.sender(conn, srcAddr, s.makeAttrs(m.TransactionID, stun.NewType(stun.MethodChannelBind, stun.ClassSuccessResponse), messageIntegrity)...)
}

func (s *Server) handleChannelData(conn net.PacketConn, srcAddr net.Addr, c *proto.ChannelData) error {
	s.log.Debugf("received ChannelData from %s", srcAddr.String())
	dstAddr := conn.LocalAddr()
	a := s.manager.GetAllocation(&allocation.FiveTuple{
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
		Protocol: allocation.UDP,
	})
	if a == nil {
		return errors.Errorf("No allocation found for %v:%v", srcAddr, dstAddr)
	}

	channel := a.GetChannelByNumber(c.Number)
	if channel == nil {
		return errors.Errorf("No channel bind found for %x \n", uint16(c.Number))
	}

	l, err := a.RelaySocket.WriteTo(c.Data, channel.Peer)
	if err != nil {
		return errors.Wrap(err, "failed writing to socket")
	}

	if l != len(c.Data) {
		return errors.Errorf("packet write smaller than packet %d != %d (expected)", l, len(c.Data))
	}

	return nil
}

func (s *Server) makeAttrs(transactionID [stun.TransactionIDSize]byte, msgType stun.MessageType, additional ...stun.Setter) []stun.Setter {
	attrs := append([]stun.Setter{&stun.Message{TransactionID: transactionID}, msgType}, additional...)
	if len(s.software) > 0 {
		attrs = append(attrs, s.software)
	}
	return attrs
}

func allocationLifeTime(m *stun.Message) time.Duration {
	lifetimeDuration := proto.DefaultLifetime

	var lifetime proto.Lifetime
	if err := lifetime.GetFrom(m); err == nil {
		if lifetime.Duration < maximumLifetime {
			lifetimeDuration = lifetime.Duration
		}
	}

	return lifetimeDuration
}
