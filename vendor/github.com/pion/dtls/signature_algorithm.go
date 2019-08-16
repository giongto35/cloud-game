package dtls

// https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-16
type signatureAlgorithm uint16

const (
	signatureAlgorithmRSA   signatureAlgorithm = 1
	signatureAlgorithmECDSA signatureAlgorithm = 3
)

var signatureAlgorithms = map[signatureAlgorithm]bool{
	signatureAlgorithmRSA:   true,
	signatureAlgorithmECDSA: true,
}
