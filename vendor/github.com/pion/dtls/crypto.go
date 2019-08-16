package dtls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"math/big"
)

type ecdsaSignature struct {
	R, S *big.Int
}

func valueKeySignature(clientRandom, serverRandom, publicKey []byte, namedCurve namedCurve, hashAlgorithm HashAlgorithm) []byte {
	serverECDHParams := make([]byte, 4)
	serverECDHParams[0] = 3 // named curve
	binary.BigEndian.PutUint16(serverECDHParams[1:], uint16(namedCurve))
	serverECDHParams[3] = byte(len(publicKey))

	plaintext := []byte{}
	plaintext = append(plaintext, clientRandom...)
	plaintext = append(plaintext, serverRandom...)
	plaintext = append(plaintext, serverECDHParams...)
	plaintext = append(plaintext, publicKey...)
	return hashAlgorithm.digest(plaintext)
}

// If the client provided a "signature_algorithms" extension, then all
// certificates provided by the server MUST be signed by a
// hash/signature algorithm pair that appears in that extension
//
// https://tools.ietf.org/html/rfc5246#section-7.4.2
func generateKeySignature(clientRandom, serverRandom, publicKey []byte, namedCurve namedCurve, privateKey crypto.PrivateKey, hashAlgorithm HashAlgorithm) ([]byte, error) {
	hashed := valueKeySignature(clientRandom, serverRandom, publicKey, namedCurve, hashAlgorithm)
	switch p := privateKey.(type) {
	case *ecdsa.PrivateKey:
		return p.Sign(rand.Reader, hashed, crypto.SHA256)
	case *rsa.PrivateKey:
		return p.Sign(rand.Reader, hashed, crypto.SHA256)
	}

	return nil, errKeySignatureGenerateUnimplemented
}

func verifyKeySignature(hash, remoteKeySignature []byte, hashAlgorithm HashAlgorithm, certificate *x509.Certificate) error {
	switch p := certificate.PublicKey.(type) {
	case *ecdsa.PublicKey:
		ecdsaSig := &ecdsaSignature{}
		if _, err := asn1.Unmarshal(remoteKeySignature, ecdsaSig); err != nil {
			return err
		}
		if ecdsaSig.R.Sign() <= 0 || ecdsaSig.S.Sign() <= 0 {
			return errInvalidECDSASignature
		}
		if !ecdsa.Verify(p, hash, ecdsaSig.R, ecdsaSig.S) {
			return errKeySignatureMismatch
		}
		return nil
	case *rsa.PublicKey:
		switch certificate.SignatureAlgorithm {
		case x509.SHA1WithRSA, x509.SHA256WithRSA, x509.SHA384WithRSA, x509.SHA512WithRSA:
			return rsa.VerifyPKCS1v15(p, hashAlgorithm.cryptoHash(), hash, remoteKeySignature)
		}
	}

	return errKeySignatureVerifyUnimplemented
}

// If the server has sent a CertificateRequest message, the client MUST send the Certificate
// message.  The ClientKeyExchange message is now sent, and the content
// of that message will depend on the public key algorithm selected
// between the ClientHello and the ServerHello.  If the client has sent
// a certificate with signing ability, a digitally-signed
// CertificateVerify message is sent to explicitly verify possession of
// the private key in the certificate.
// https://tools.ietf.org/html/rfc5246#section-7.3
func generateCertificateVerify(handshakeBodies []byte, privateKey crypto.PrivateKey) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(handshakeBodies); err != nil {
		return nil, err
	}
	hashed := h.Sum(nil)

	switch p := privateKey.(type) {
	case *ecdsa.PrivateKey:
		return p.Sign(rand.Reader, hashed, crypto.SHA256)
	case *rsa.PrivateKey:
		return p.Sign(rand.Reader, hashed, crypto.SHA256)
	}

	return nil, errInvalidSignatureAlgorithm
}

func verifyCertificateVerify(handshakeBodies []byte, hashAlgorithm HashAlgorithm, remoteKeySignature []byte, certificate *x509.Certificate) error {
	hash := hashAlgorithm.digest(handshakeBodies)
	switch p := certificate.PublicKey.(type) {
	case *ecdsa.PublicKey:
		ecdsaSig := &ecdsaSignature{}
		if _, err := asn1.Unmarshal(remoteKeySignature, ecdsaSig); err != nil {
			return err
		}
		if ecdsaSig.R.Sign() <= 0 || ecdsaSig.S.Sign() <= 0 {
			return errInvalidECDSASignature
		}
		if !ecdsa.Verify(p, hash, ecdsaSig.R, ecdsaSig.S) {
			return errKeySignatureMismatch
		}
		return nil
	case *rsa.PublicKey:
		switch certificate.SignatureAlgorithm {
		case x509.SHA1WithRSA, x509.SHA256WithRSA, x509.SHA384WithRSA, x509.SHA512WithRSA:
			return rsa.VerifyPKCS1v15(p, hashAlgorithm.cryptoHash(), hash, remoteKeySignature)
		}
	}

	return errKeySignatureVerifyUnimplemented
}
