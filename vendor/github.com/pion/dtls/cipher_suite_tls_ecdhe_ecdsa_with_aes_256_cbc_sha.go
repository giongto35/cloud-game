package dtls

import (
	"crypto/sha256"
	"errors"
	"hash"
)

type cipherSuiteTLSEcdheEcdsaWithAes256CbcSha struct {
	cbc *cryptoCBC
}

func (c cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) certificateType() clientCertificateType {
	return clientCertificateTypeECDSASign
}

func (c cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) ID() CipherSuiteID {
	return TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
}

func (c cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) String() string {
	return "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA"
}

func (c cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) hashFunc() func() hash.Hash {
	return sha256.New
}

func (c cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) isPSK() bool {
	return false
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) init(masterSecret, clientRandom, serverRandom []byte, isClient bool) error {
	const (
		prfMacLen = 20
		prfKeyLen = 32
		prfIvLen  = 16
	)

	keys, err := prfEncryptionKeys(masterSecret, clientRandom, serverRandom, prfMacLen, prfKeyLen, prfIvLen, c.hashFunc())
	if err != nil {
		return err
	}

	if isClient {
		c.cbc, err = newCryptoCBC(
			keys.clientWriteKey, keys.clientWriteIV, keys.clientMACKey,
			keys.serverWriteKey, keys.serverWriteIV, keys.serverMACKey,
		)
	} else {
		c.cbc, err = newCryptoCBC(
			keys.serverWriteKey, keys.serverWriteIV, keys.serverMACKey,
			keys.clientWriteKey, keys.clientWriteIV, keys.clientMACKey,
		)
	}

	return err
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) encrypt(pkt *recordLayer, raw []byte) ([]byte, error) {
	if c.cbc == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to encrypt")
	}

	return c.cbc.encrypt(pkt, raw)
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes256CbcSha) decrypt(raw []byte) ([]byte, error) {
	if c.cbc == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to decrypt ")
	}

	return c.cbc.decrypt(raw)
}
