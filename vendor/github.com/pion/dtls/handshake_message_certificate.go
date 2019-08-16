package dtls

import (
	"crypto/x509"
)

type handshakeMessageCertificate struct {
	certificate *x509.Certificate
}

func (h handshakeMessageCertificate) handshakeType() handshakeType {
	return handshakeTypeCertificate
}

func (h *handshakeMessageCertificate) Marshal() ([]byte, error) {
	if h.certificate == nil {
		return nil, errCertificateUnset
	}

	out := make([]byte, 6)
	putBigEndianUint24(out, uint32(len(h.certificate.Raw))+3)
	putBigEndianUint24(out[3:], uint32(len(h.certificate.Raw)))

	return append(out, h.certificate.Raw...), nil
}

func (h *handshakeMessageCertificate) Unmarshal(data []byte) error {
	if len(data) < 6 {
		return errBufferTooSmall
	}

	certificateBodyLen := int(bigEndianUint24(data))
	certificateLen := int(bigEndianUint24(data[3:]))
	if certificateBodyLen+3 != len(data) {
		return errLengthMismatch
	} else if certificateLen+6 != len(data) {
		return errLengthMismatch
	}

	cert, err := x509.ParseCertificate(data[6:])
	if err != nil {
		return err
	}
	h.certificate = cert

	return nil
}
