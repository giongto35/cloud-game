package dtls

import (
	"crypto"
	"crypto/md5"  // #nosec
	"crypto/sha1" // #nosec
	"crypto/sha256"
	"crypto/sha512"
)

// HashAlgorithm is used to indicate the hash algorithm used
// https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-18
type HashAlgorithm uint16

// Supported hash hash algorithms
const (
	// HashAlgorithmMD2    HashAlgorithm = 0 // Blacklisted
	HashAlgorithmMD5    HashAlgorithm = 1 // Blacklisted
	HashAlgorithmSHA1   HashAlgorithm = 2 // Blacklisted
	HashAlgorithmSHA224 HashAlgorithm = 3
	HashAlgorithmSHA256 HashAlgorithm = 4
	HashAlgorithmSHA384 HashAlgorithm = 5
	HashAlgorithmSHA512 HashAlgorithm = 6
)

// String makes HashAlgorithm printable
func (h HashAlgorithm) String() string {
	switch h {
	case HashAlgorithmMD5:
		return "md5" // [RFC3279]
	case HashAlgorithmSHA1:
		return "sha-1" // [RFC3279]
	case HashAlgorithmSHA224:
		return "sha-224" // [RFC4055]
	case HashAlgorithmSHA256:
		return "sha-256" // [RFC4055]
	case HashAlgorithmSHA384:
		return "sha-384" // [RFC4055]
	case HashAlgorithmSHA512:
		return "sha-512" // [RFC4055]
	default:
		return "unknown hash algorithm"
	}
}

// HashAlgorithmString allows looking up a HashAlgorithm by it's string representation
func HashAlgorithmString(s string) (HashAlgorithm, error) {
	switch s {
	case "md5":
		return HashAlgorithmMD5, nil // [RFC3279]
	case "sha-1":
		return HashAlgorithmSHA1, nil // [RFC3279]
	case "sha-224":
		return HashAlgorithmSHA224, nil // [RFC4055]
	case "sha-256":
		return HashAlgorithmSHA256, nil // [RFC4055]
	case "sha-384":
		return HashAlgorithmSHA384, nil // [RFC4055]
	case "sha-512":
		return HashAlgorithmSHA512, nil // [RFC4055]
	default:
		return 0, errInvalidHashAlgorithm
	}
}

func (h HashAlgorithm) digest(b []byte) []byte {
	switch h {
	case HashAlgorithmMD5:
		hash := md5.Sum(b) // #nosec
		return hash[:]
	case HashAlgorithmSHA1:
		hash := sha1.Sum(b) // #nosec
		return hash[:]
	case HashAlgorithmSHA224:
		hash := sha256.Sum224(b)
		return hash[:]
	case HashAlgorithmSHA256:
		hash := sha256.Sum256(b)
		return hash[:]
	case HashAlgorithmSHA384:
		hash := sha512.Sum384(b)
		return hash[:]
	case HashAlgorithmSHA512:
		hash := sha512.Sum512(b)
		return hash[:]
	default:
		return nil
	}
}

func (h HashAlgorithm) cryptoHash() crypto.Hash {
	switch h {
	case HashAlgorithmMD5:
		return crypto.MD5
	case HashAlgorithmSHA1:
		return crypto.SHA1
	case HashAlgorithmSHA224:
		return crypto.SHA224
	case HashAlgorithmSHA256:
		return crypto.SHA256
	case HashAlgorithmSHA384:
		return crypto.SHA384
	case HashAlgorithmSHA512:
		return crypto.SHA512
	default:
		return 0
	}
}

var hashAlgorithms = map[HashAlgorithm]struct{}{
	HashAlgorithmMD5:    {},
	HashAlgorithmSHA1:   {},
	HashAlgorithmSHA224: {},
	HashAlgorithmSHA256: {},
	HashAlgorithmSHA384: {},
	HashAlgorithmSHA512: {},
}
