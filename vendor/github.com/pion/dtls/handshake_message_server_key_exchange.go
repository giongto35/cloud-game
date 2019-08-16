package dtls

import (
	"encoding/binary"
)

// Structure only supports ECDH
type handshakeMessageServerKeyExchange struct {
	ellipticCurveType  ellipticCurveType
	namedCurve         namedCurve
	publicKey          []byte
	hashAlgorithm      HashAlgorithm
	signatureAlgorithm signatureAlgorithm
	signature          []byte
}

func (h handshakeMessageServerKeyExchange) handshakeType() handshakeType {
	return handshakeTypeServerKeyExchange
}

func (h *handshakeMessageServerKeyExchange) Marshal() ([]byte, error) {
	out := []byte{byte(h.ellipticCurveType), 0x00, 0x00}
	binary.BigEndian.PutUint16(out[1:], uint16(h.namedCurve))

	out = append(out, byte(len(h.publicKey)))
	out = append(out, h.publicKey...)

	out = append(out, []byte{byte(h.hashAlgorithm), byte(h.signatureAlgorithm), 0x00, 0x00}...)

	binary.BigEndian.PutUint16(out[len(out)-2:], uint16(len(h.signature)))
	out = append(out, h.signature...)

	return out, nil
}

func (h *handshakeMessageServerKeyExchange) Unmarshal(data []byte) error {
	if len(data) < 1 {
		return errBufferTooSmall
	}
	if _, ok := ellipticCurveTypes[ellipticCurveType(data[0])]; ok {
		h.ellipticCurveType = ellipticCurveType(data[0])
	} else {
		return errInvalidEllipticCurveType
	}

	h.namedCurve = namedCurve(binary.BigEndian.Uint16(data[1:]))
	if _, ok := namedCurves[h.namedCurve]; !ok {
		return errInvalidNamedCurve
	}
	if len(data) < 4 {
		return errBufferTooSmall
	}

	publicKeyLength := int(data[3])
	offset := 4 + publicKeyLength
	if len(data) <= publicKeyLength {
		return errBufferTooSmall
	}
	h.publicKey = append([]byte{}, data[4:offset]...)

	h.hashAlgorithm = HashAlgorithm(data[offset])
	if _, ok := hashAlgorithms[h.hashAlgorithm]; !ok {
		return errInvalidHashAlgorithm
	}
	offset++

	h.signatureAlgorithm = signatureAlgorithm(data[offset])
	if _, ok := signatureAlgorithms[h.signatureAlgorithm]; !ok {
		return errInvalidSignatureAlgorithm
	}
	offset++

	signatureLength := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2
	h.signature = append([]byte{}, data[offset:offset+signatureLength]...)
	return nil
}
