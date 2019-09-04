package dtls

import (
	"crypto/x509"
	"errors"
	"fmt"
)

// Fingerprint creates a fingerprint for a certificate using the specified hash algorithm
func Fingerprint(cert *x509.Certificate, algo HashAlgorithm) (string, error) {
	digest := []byte(fmt.Sprintf("%x", algo.digest(cert.Raw)))

	digestlen := len(digest)
	if digestlen == 0 {
		return "", nil
	}
	if digestlen%2 != 0 {
		return "", errors.New("invalid fingerprint length")
	}
	res := make([]byte, digestlen>>1+digestlen-1)

	pos := 0
	for i, c := range digest {
		res[pos] = c
		pos++
		if (i)%2 != 0 && i < digestlen-1 {
			res[pos] = byte(':')
			pos++
		}
	}

	return string(res), nil
}
