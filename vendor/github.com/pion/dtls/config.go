package dtls

import (
	"crypto"
	"crypto/x509"
	"time"

	"github.com/pion/logging"
)

// Config is used to configure a DTLS client or server.
// After a Config is passed to a DTLS function it must not be modified.
type Config struct {
	// Certificates contains certificate chain to present to the other side of the connection.
	// Server MUST set this if PSK is non-nil
	// client SHOULD sets this so CertificateRequests can be handled if PSK is non-nil
	Certificate *x509.Certificate

	// PrivateKey contains matching private key for the certificate
	// only ECDSA is supported
	PrivateKey crypto.PrivateKey

	// CipherSuites is a list of supported cipher suites.
	// If CipherSuites is nil, a default list is used
	CipherSuites []CipherSuiteID

	// SRTPProtectionProfiles are the supported protection profiles
	// Clients will send this via use_srtp and assert that the server properly responds
	// Servers will assert that clients send one of these profiles and will respond as needed
	SRTPProtectionProfiles []SRTPProtectionProfile

	// ClientAuth determines the server's policy for
	// TLS Client Authentication. The default is NoClientCert.
	ClientAuth ClientAuthType

	// FlightInterval controls how often we send outbound handshake messages
	// defaults to time.Second
	FlightInterval time.Duration

	// PSK sets the pre-shared key used by this DTLS connection
	// If PSK is non-nil only PSK CipherSuites will be used
	PSK             PSKCallback
	PSKIdentityHint []byte

	LoggerFactory logging.LoggerFactory
}

// PSKCallback is called once we have the remote's PSKIdentityHint.
// If the remote provided none it will be nil
type PSKCallback func([]byte) ([]byte, error)

// ClientAuthType declares the policy the server will follow for
// TLS Client Authentication.
type ClientAuthType int

// ClientAuthType enums
const (
	NoClientCert ClientAuthType = iota
	RequestClientCert
	RequireAnyClientCert
)
