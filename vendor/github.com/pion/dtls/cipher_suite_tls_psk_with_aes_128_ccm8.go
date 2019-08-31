package dtls

import (
	"crypto/sha256"
	"errors"
	"hash"
)

type cipherSuiteTLSPskWithAes128Ccm8 struct {
	ccm *cryptoCCM
}

func (c cipherSuiteTLSPskWithAes128Ccm8) certificateType() clientCertificateType {
	return clientCertificateType(0)
}

func (c cipherSuiteTLSPskWithAes128Ccm8) ID() CipherSuiteID {
	return TLS_PSK_WITH_AES_128_CCM_8
}

func (c cipherSuiteTLSPskWithAes128Ccm8) String() string {
	return "TLS_PSK_WITH_AES_128_CCM_8"
}

func (c cipherSuiteTLSPskWithAes128Ccm8) hashFunc() func() hash.Hash {
	return sha256.New
}

func (c cipherSuiteTLSPskWithAes128Ccm8) isPSK() bool {
	return true
}

func (c *cipherSuiteTLSPskWithAes128Ccm8) init(masterSecret, clientRandom, serverRandom []byte, isClient bool) error {
	const (
		prfMacLen = 0
		prfKeyLen = 16
		prfIvLen  = 4
	)

	keys, err := prfEncryptionKeys(masterSecret, clientRandom, serverRandom, prfMacLen, prfKeyLen, prfIvLen, c.hashFunc())
	if err != nil {
		return err
	}

	if isClient {
		c.ccm, err = newCryptoCCM(keys.clientWriteKey, keys.clientWriteIV, keys.serverWriteKey, keys.serverWriteIV)
	} else {
		c.ccm, err = newCryptoCCM(keys.serverWriteKey, keys.serverWriteIV, keys.clientWriteKey, keys.clientWriteIV)
	}
	return err
}

func (c *cipherSuiteTLSPskWithAes128Ccm8) encrypt(pkt *recordLayer, raw []byte) ([]byte, error) {
	if c.ccm == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to encrypt")
	}

	return c.ccm.encrypt(pkt, raw)
}

func (c *cipherSuiteTLSPskWithAes128Ccm8) decrypt(raw []byte) ([]byte, error) {
	if c.ccm == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to decrypt ")
	}

	return c.ccm.decrypt(raw)
}
