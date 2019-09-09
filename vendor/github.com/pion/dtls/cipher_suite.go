package dtls

import (
	"encoding/binary"
	"fmt"
	"hash"
)

// CipherSuiteID is an ID for our supported CipherSuites
type CipherSuiteID uint16

// Supported Cipher Suites
const (
	// AES-128-GCM-SHA256
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 CipherSuiteID = 0xc02b
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256   CipherSuiteID = 0xc02f

	// AES-256-CBC-SHA
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA CipherSuiteID = 0xc00a
	TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA   CipherSuiteID = 0x0035

	TLS_PSK_WITH_AES_128_CCM_8      CipherSuiteID = 0xc0a8
	TLS_PSK_WITH_AES_128_GCM_SHA256 CipherSuiteID = 0x00a8
)

type cipherSuite interface {
	String() string
	ID() CipherSuiteID
	certificateType() clientCertificateType
	hashFunc() func() hash.Hash
	isPSK() bool

	// Generate the internal encryption state
	init(masterSecret, clientRandom, serverRandom []byte, isClient bool) error

	encrypt(pkt *recordLayer, raw []byte) ([]byte, error)
	decrypt(in []byte) ([]byte, error)
}

// Taken from https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
// A cipherSuite is a specific combination of key agreement, cipher and MAC
// function.
func cipherSuiteForID(id CipherSuiteID) cipherSuite {
	switch id {
	case cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256{}.ID():
		return &cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256{}
	case cipherSuiteTLSEcdheRsaWithAes128GcmSha256{}.ID():
		return &cipherSuiteTLSEcdheRsaWithAes128GcmSha256{}
	case cipherSuiteTLSEcdheEcdsaWithAes256CbcSha{}.ID():
		return &cipherSuiteTLSEcdheEcdsaWithAes256CbcSha{}
	case cipherSuiteTLSEcdheRsaWithAes256CbcSha{}.ID():
		return &cipherSuiteTLSEcdheRsaWithAes256CbcSha{}
	case cipherSuiteTLSPskWithAes128Ccm8{}.ID():
		return &cipherSuiteTLSPskWithAes128Ccm8{}
	case cipherSuiteTLSPskWithAes128GcmSha256{}.ID():
		return &cipherSuiteTLSPskWithAes128GcmSha256{}
	}
	return nil
}

// CipherSuites we support in order of preference
func defaultCipherSuites() []cipherSuite {
	return []cipherSuite{
		&cipherSuiteTLSEcdheRsaWithAes256CbcSha{},
		&cipherSuiteTLSEcdheEcdsaWithAes256CbcSha{},
		&cipherSuiteTLSEcdheRsaWithAes128GcmSha256{},
		&cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256{},
	}
}

func decodeCipherSuites(buf []byte) ([]cipherSuite, error) {
	if len(buf) < 2 {
		return nil, errDTLSPacketInvalidLength
	}
	cipherSuitesCount := int(binary.BigEndian.Uint16(buf[0:])) / 2
	rtrn := []cipherSuite{}
	for i := 0; i < cipherSuitesCount; i++ {
		if len(buf) < (i*2 + 4) {
			return nil, errBufferTooSmall
		}
		id := CipherSuiteID(binary.BigEndian.Uint16(buf[(i*2)+2:]))
		if c := cipherSuiteForID(id); c != nil {
			rtrn = append(rtrn, c)
		}
	}
	return rtrn, nil
}

func encodeCipherSuites(c []cipherSuite) []byte {
	out := []byte{0x00, 0x00}
	binary.BigEndian.PutUint16(out[len(out)-2:], uint16(len(c)*2))
	for i := len(c); i > 0; i-- {
		out = append(out, []byte{0x00, 0x00}...)
		binary.BigEndian.PutUint16(out[len(out)-2:], uint16(c[i-1].ID()))
	}

	return out
}

func parseCipherSuites(userSelectedSuites []CipherSuiteID, excludePSK, excludeNonPSK bool) ([]cipherSuite, error) {
	cipherSuitesForIDs := func(ids []CipherSuiteID) ([]cipherSuite, error) {
		cipherSuites := []cipherSuite{}
		for _, id := range ids {
			c := cipherSuiteForID(id)
			if c == nil {
				return nil, fmt.Errorf("CipherSuite with id(%d) is not valid", id)
			}
			cipherSuites = append(cipherSuites, c)
		}
		return cipherSuites, nil
	}

	var (
		cipherSuites []cipherSuite
		err          error
		i            int
	)
	if len(userSelectedSuites) != 0 {
		cipherSuites, err = cipherSuitesForIDs(userSelectedSuites)
		if err != nil {
			return nil, err
		}
	} else {
		cipherSuites = defaultCipherSuites()
	}

	for _, c := range cipherSuites {
		if excludePSK && c.isPSK() || excludeNonPSK && !c.isPSK() {
			continue
		}
		cipherSuites[i] = c
		i++
	}

	cipherSuites = cipherSuites[:i]
	if len(cipherSuites) == 0 {
		return nil, errNoAvailableCipherSuites
	}

	return cipherSuites, nil
}
