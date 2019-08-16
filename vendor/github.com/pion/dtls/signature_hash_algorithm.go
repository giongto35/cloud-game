package dtls

type signatureHashAlgorithm struct {
	hash      HashAlgorithm
	signature signatureAlgorithm
}
