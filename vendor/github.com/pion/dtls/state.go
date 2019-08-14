package dtls

import (
	"bytes"
	"crypto/x509"
	"encoding/gob"
	"sync/atomic"
)

// State holds the dtls connection state and implements both encoding.BinaryMarshaler and encoding.BinaryUnmarshaler
type State struct {
	localEpoch, remoteEpoch   atomic.Value
	localSequenceNumber       uint64 // uint48
	localRandom, remoteRandom handshakeRandom
	masterSecret              []byte
	cipherSuite               cipherSuite // nil if a cipherSuite hasn't been chosen

	srtpProtectionProfile SRTPProtectionProfile // Negotiated SRTPProtectionProfile
	remoteCertificate     *x509.Certificate

	isClient bool
}

type serializedState struct {
	LocalEpoch            uint16
	RemoteEpoch           uint16
	LocalRandom           []byte
	RemoteRandom          []byte
	CipherSuiteID         uint16
	MasterSecret          []byte
	SequenceNumber        uint64
	SRTPProtectionProfile uint16
	RemoteCertificate     []byte
	IsClient              bool
}

func (s *State) clone() (*State, error) {
	serialized, err := s.serialize()
	if err != nil {
		return nil, err
	}
	state := &State{}
	if err := state.deserialize(*serialized); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) serialize() (*serializedState, error) {
	// Marshal random values
	localRnd, err := s.localRandom.Marshal()
	if err != nil {
		return nil, err
	}
	remoteRnd, err := s.remoteRandom.Marshal()
	if err != nil {
		return nil, err
	}

	// Marshal remote certificate
	var cert []byte
	if s.remoteCertificate != nil {
		h := &handshakeMessageCertificate{s.remoteCertificate}
		cert, err = h.Marshal()
		if err != nil {
			return nil, err
		}
	}

	serialized := serializedState{
		LocalEpoch:            s.localEpoch.Load().(uint16),
		RemoteEpoch:           s.remoteEpoch.Load().(uint16),
		CipherSuiteID:         uint16(s.cipherSuite.ID()),
		MasterSecret:          s.masterSecret,
		SequenceNumber:        s.localSequenceNumber,
		LocalRandom:           localRnd,
		RemoteRandom:          remoteRnd,
		SRTPProtectionProfile: uint16(s.srtpProtectionProfile),
		RemoteCertificate:     cert,
		IsClient:              s.isClient,
	}

	return &serialized, nil
}

func (s *State) deserialize(serialized serializedState) error {
	// Set epoch values
	s.localEpoch.Store(serialized.LocalEpoch)
	s.remoteEpoch.Store(serialized.RemoteEpoch)

	// Set random values
	localRandom := &handshakeRandom{}
	if err := localRandom.Unmarshal(serialized.LocalRandom); err != nil {
		return err
	}
	s.localRandom = *localRandom
	remoteRandom := &handshakeRandom{}
	if err := remoteRandom.Unmarshal(serialized.RemoteRandom); err != nil {
		return err
	}
	s.remoteRandom = *remoteRandom

	s.isClient = serialized.IsClient

	// Set cipher suite
	s.cipherSuite = cipherSuiteForID(CipherSuiteID(serialized.CipherSuiteID))
	var err error
	if serialized.IsClient {
		err = s.cipherSuite.init(serialized.MasterSecret, serialized.LocalRandom, serialized.RemoteRandom, true)
	} else {
		err = s.cipherSuite.init(serialized.MasterSecret, serialized.RemoteRandom, serialized.LocalRandom, false)
	}
	if err != nil {
		return err
	}

	s.localSequenceNumber = serialized.SequenceNumber
	s.srtpProtectionProfile = SRTPProtectionProfile(serialized.SRTPProtectionProfile)

	// Set remote certificate
	if serialized.RemoteCertificate != nil {
		h := &handshakeMessageCertificate{}
		if err := h.Unmarshal(serialized.RemoteCertificate); err != nil {
			return err
		}
		s.remoteCertificate = h.certificate
	}

	return nil
}

// MarshalBinary is a binary.BinaryMarshaler.MarshalBinary implementation
func (s *State) MarshalBinary() ([]byte, error) {
	serialized, err := s.serialize()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*serialized); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary is a binary.BinaryUnmarshaler.UnmarshalBinary implementation
func (s *State) UnmarshalBinary(data []byte) error {
	enc := gob.NewDecoder(bytes.NewBuffer(data))
	var serialized serializedState
	if err := enc.Decode(&serialized); err != nil {
		return err
	}

	if err := s.deserialize(serialized); err != nil {
		return err
	}
	return nil
}
