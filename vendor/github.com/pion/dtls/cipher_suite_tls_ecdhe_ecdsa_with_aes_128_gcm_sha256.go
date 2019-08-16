package dtls

import (
	"crypto/sha256"
	"errors"
	"hash"
)

type cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256 struct {
	gcm *cryptoGCM
}

func (c cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) certificateType() clientCertificateType {
	return clientCertificateTypeECDSASign
}

func (c cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) ID() CipherSuiteID {
	return TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
}

func (c cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) String() string {
	return "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
}

func (c cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) hashFunc() func() hash.Hash {
	return sha256.New
}

func (c cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) isPSK() bool {
	return false
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) init(masterSecret, clientRandom, serverRandom []byte, isClient bool) error {
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
		c.gcm, err = newCryptoGCM(keys.clientWriteKey, keys.clientWriteIV, keys.serverWriteKey, keys.serverWriteIV)
	} else {
		c.gcm, err = newCryptoGCM(keys.serverWriteKey, keys.serverWriteIV, keys.clientWriteKey, keys.clientWriteIV)
	}

	return err
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) encrypt(pkt *recordLayer, raw []byte) ([]byte, error) {
	if c.gcm == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to encrypt")
	}

	return c.gcm.encrypt(pkt, raw)
}

func (c *cipherSuiteTLSEcdheEcdsaWithAes128GcmSha256) decrypt(raw []byte) ([]byte, error) {
	if c.gcm == nil {
		return nil, errors.New("CipherSuite has not been initalized, unable to decrypt ")
	}

	return c.gcm.decrypt(raw)
}
