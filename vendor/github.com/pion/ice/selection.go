package ice

import (
	"net"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun"
)

type pairCandidateSelector interface {
	Start()
	ContactCandidates()
	PingCandidate(local, remote Candidate)
	HandleSucessResponse(m *stun.Message, local, remote Candidate, remoteAddr net.Addr)
	HandleBindingRequest(m *stun.Message, local, remote Candidate)
}

type controllingSelector struct {
	startTime              time.Time
	agent                  *Agent
	nominatedPair          *candidatePair
	nominationRequestCount uint16
	log                    logging.LeveledLogger
}

func (s *controllingSelector) Start() {
	s.startTime = time.Now()
	go func() {
		time.Sleep(s.agent.candidateSelectionTimeout)
		err := s.agent.run(func(a *Agent) {
			if s.nominatedPair == nil {
				p := s.agent.getBestValidCandidatePair()
				if p == nil {
					s.log.Trace("check timeout reached and no valid candidate pair found, marking connection as failed")
					s.agent.updateConnectionState(ConnectionStateFailed)
				} else {
					s.log.Tracef("check timeout reached, nominating (%s, %s)", p.local.String(), p.remote.String())
					s.nominatedPair = p
					s.nominatePair(p)
				}
			}
		})

		if err != nil {
			s.log.Errorf("error processing checkCandidatesTimeout handler %v", err.Error())
		}
	}()
}

func (s *controllingSelector) isNominatable(c Candidate) bool {
	switch {
	case c.Type() == CandidateTypeHost:
		return time.Since(s.startTime).Nanoseconds() > s.agent.hostAcceptanceMinWait.Nanoseconds()
	case c.Type() == CandidateTypeServerReflexive:
		return time.Since(s.startTime).Nanoseconds() > s.agent.srflxAcceptanceMinWait.Nanoseconds()
	case c.Type() == CandidateTypePeerReflexive:
		return time.Since(s.startTime).Nanoseconds() > s.agent.prflxAcceptanceMinWait.Nanoseconds()
	case c.Type() == CandidateTypeRelay:
		return time.Since(s.startTime).Nanoseconds() > s.agent.relayAcceptanceMinWait.Nanoseconds()
	}

	s.log.Errorf("isNominatable invalid candidate type %s", c.Type().String())
	return false
}

func (s *controllingSelector) ContactCandidates() {
	switch {
	case s.agent.selectedPair != nil:
		if s.agent.validateSelectedPair() {
			s.log.Trace("checking keepalive")
			s.agent.checkKeepalive()
		}
	case s.nominatedPair != nil:
		if s.nominationRequestCount > s.agent.maxBindingRequests {
			s.log.Trace("max nomination requests reached, setting the connection state to failed")
			s.agent.updateConnectionState(ConnectionStateFailed)
			return
		}
		s.nominatePair(s.nominatedPair)
	default:
		p := s.agent.getBestValidCandidatePair()
		if p != nil && s.isNominatable(p.local) && s.isNominatable(p.remote) {
			s.log.Tracef("Nominatable pair found, nominating (%s, %s)", p.local.String(), p.remote.String())
			p.nominated = true
			s.nominatedPair = p
			s.nominatePair(p)
			return
		}

		s.log.Trace("pinging all candidates")
		s.agent.pingAllCandidates()
	}
}

func (s *controllingSelector) nominatePair(pair *candidatePair) {
	// The controlling agent MUST include the USE-CANDIDATE attribute in
	// order to nominate a candidate pair (Section 8.1.1).  The controlled
	// agent MUST NOT include the USE-CANDIDATE attribute in a Binding
	// request.
	msg, err := stun.Build(stun.BindingRequest, stun.TransactionID,
		stun.NewUsername(s.agent.remoteUfrag+":"+s.agent.localUfrag),
		UseCandidate,
		AttrControlling(s.agent.tieBreaker),
		PriorityAttr(pair.local.Priority()),
		stun.NewShortTermIntegrity(s.agent.remotePwd),
		stun.Fingerprint,
	)

	if err != nil {
		s.log.Error(err.Error())
		return
	}

	s.log.Tracef("ping STUN (nominate candidate pair) from %s to %s\n", pair.local.String(), pair.remote.String())
	s.agent.sendBindingRequest(msg, pair.local, pair.remote)
	s.nominationRequestCount++
}

func (s *controllingSelector) HandleBindingRequest(m *stun.Message, local, remote Candidate) {
	s.agent.sendBindingSuccess(m, local, remote)

	p := s.agent.findPair(local, remote)

	if p == nil {
		s.agent.addPair(local, remote)
		return
	}

	if p.state == CandidatePairStateSucceeded && s.nominatedPair == nil && s.agent.selectedPair == nil {
		bestPair := s.agent.getBestAvailableCandidatePair()
		if bestPair == nil {
			s.log.Tracef("No best pair available\n")
		} else if bestPair.Equal(p) && s.isNominatable(p.local) && s.isNominatable(p.remote) {
			s.log.Tracef("The candidate (%s, %s) is the best candidate available, marking it as nominated\n",
				p.local.String(), p.remote.String())
			s.nominatedPair = p
			s.nominatePair(p)
		}
	}
}

func (s *controllingSelector) HandleSucessResponse(m *stun.Message, local, remote Candidate, remoteAddr net.Addr) {
	ok, pendingRequest := s.agent.handleInboundBindingSuccess(m.TransactionID)
	if !ok {
		s.log.Warnf("discard message from (%s), unknown TransactionID 0x%x", remote, m.TransactionID)
		return
	}

	transactionAddr := pendingRequest.destination

	// Assert that NAT is not symmetric
	// https://tools.ietf.org/html/rfc8445#section-7.2.5.2.1
	if !addrEqual(transactionAddr, remoteAddr) {
		s.log.Debugf("discard message: transaction source and destination does not match expected(%s), actual(%s)", transactionAddr, remote)
		return
	}

	s.log.Tracef("inbound STUN (SuccessResponse) from %s to %s", remote.String(), local.String())
	p := s.agent.findPair(local, remote)

	if p == nil {
		// This shouldn't happen
		s.log.Error("Success response from invalid candidate pair")
		return
	}

	p.state = CandidatePairStateSucceeded
	s.log.Tracef("Found valid candidate pair: %s", p)
	if pendingRequest.isUseCandidate && s.agent.selectedPair == nil {
		s.agent.setSelectedPair(p)
	}
}

func (s *controllingSelector) PingCandidate(local, remote Candidate) {
	msg, err := stun.Build(stun.BindingRequest, stun.TransactionID,
		stun.NewUsername(s.agent.remoteUfrag+":"+s.agent.localUfrag),
		AttrControlling(s.agent.tieBreaker),
		PriorityAttr(local.Priority()),
		stun.NewShortTermIntegrity(s.agent.remotePwd),
		stun.Fingerprint,
	)

	if err != nil {
		s.log.Error(err.Error())
		return
	}

	s.agent.sendBindingRequest(msg, local, remote)
}

type controlledSelector struct {
	agent *Agent
	log   logging.LeveledLogger
}

func (s *controlledSelector) Start() {}

func (s *controlledSelector) ContactCandidates() {
	if s.agent.selectedPair != nil {
		if s.agent.validateSelectedPair() {
			s.log.Trace("checking keepalive")
			s.agent.checkKeepalive()
		}
	} else {
		s.log.Trace("pinging all candidates")
		s.agent.pingAllCandidates()
	}
}

func (s *controlledSelector) PingCandidate(local, remote Candidate) {
	msg, err := stun.Build(stun.BindingRequest, stun.TransactionID,
		stun.NewUsername(s.agent.remoteUfrag+":"+s.agent.localUfrag),
		AttrControlled(s.agent.tieBreaker),
		PriorityAttr(local.Priority()),
		stun.NewShortTermIntegrity(s.agent.remotePwd),
		stun.Fingerprint,
	)

	if err != nil {
		s.log.Error(err.Error())
		return
	}

	s.agent.sendBindingRequest(msg, local, remote)
}

func (s *controlledSelector) HandleSucessResponse(m *stun.Message, local, remote Candidate, remoteAddr net.Addr) {
	// TODO according to the standard we should specifically answer a failed nomination:
	// https://tools.ietf.org/html/rfc8445#section-7.3.1.5
	// If the controlled agent does not accept the request from the
	// controlling agent, the controlled agent MUST reject the nomination
	// request with an appropriate error code response (e.g., 400)
	// [RFC5389].

	ok, pendingRequest := s.agent.handleInboundBindingSuccess(m.TransactionID)
	if !ok {
		s.log.Warnf("discard message from (%s), unknown TransactionID 0x%x", remote, m.TransactionID)
		return
	}

	transactionAddr := pendingRequest.destination

	// Assert that NAT is not symmetric
	// https://tools.ietf.org/html/rfc8445#section-7.2.5.2.1
	if !addrEqual(transactionAddr, remoteAddr) {
		s.log.Debugf("discard message: transaction source and destination does not match expected(%s), actual(%s)", transactionAddr, remote)
		return
	}

	s.log.Tracef("inbound STUN (SuccessResponse) from %s to %s", remote.String(), local.String())

	p := s.agent.findPair(local, remote)
	if p == nil {
		// This shouldn't happen
		s.log.Error("Success response from invalid candidate pair")
		return
	}

	p.state = CandidatePairStateSucceeded
	s.log.Tracef("Found valid candidate pair: %s", p)
}

func (s *controlledSelector) HandleBindingRequest(m *stun.Message, local, remote Candidate) {
	useCandidate := m.Contains(stun.AttrUseCandidate)

	p := s.agent.findPair(local, remote)

	if p == nil {
		p = s.agent.addPair(local, remote)
	}

	if useCandidate {
		// https://tools.ietf.org/html/rfc8445#section-7.3.1.5

		if p.state == CandidatePairStateSucceeded {
			// If the state of this pair is Succeeded, it means that the check
			// previously sent by this pair produced a successful response and
			// generated a valid pair (Section 7.2.5.3.2).  The agent sets the
			// nominated flag value of the valid pair to true.
			if s.agent.selectedPair == nil {
				s.agent.setSelectedPair(p)
			}
			s.agent.sendBindingSuccess(m, local, remote)
		} else {
			// If the received Binding request triggered a new check to be
			// enqueued in the triggered-check queue (Section 7.3.1.4), once the
			// check is sent and if it generates a successful response, and
			// generates a valid pair, the agent sets the nominated flag of the
			// pair to true.  If the request fails (Section 7.2.5.2), the agent
			// MUST remove the candidate pair from the valid list, set the
			// candidate pair state to Failed, and set the checklist state to
			// Failed.
			s.PingCandidate(local, remote)
		}
	} else {
		s.agent.sendBindingSuccess(m, local, remote)
		s.PingCandidate(local, remote)
	}
}
