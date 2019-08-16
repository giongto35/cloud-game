package dtls

type handshakeMessageClientKeyExchange struct {
	publicKey []byte
}

func (h handshakeMessageClientKeyExchange) handshakeType() handshakeType {
	return handshakeTypeClientKeyExchange
}

func (h *handshakeMessageClientKeyExchange) Marshal() ([]byte, error) {
	return append([]byte{byte(len(h.publicKey))}, h.publicKey...), nil
}

func (h *handshakeMessageClientKeyExchange) Unmarshal(data []byte) error {
	if len(data) < 1 {
		return errBufferTooSmall
	}
	publicKeyLength := int(data[0])
	if len(data) <= publicKeyLength {
		return errBufferTooSmall
	}
	h.publicKey = append([]byte{}, data[1:]...)
	return nil
}
