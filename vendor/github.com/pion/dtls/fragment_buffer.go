package dtls

type fragment struct {
	recordLayerHeader recordLayerHeader
	handshakeHeader   handshakeHeader
	data              []byte
}

type fragmentBuffer struct {
	// map of MessageSequenceNumbers that hold slices of fragments
	cache map[uint16][]*fragment

	currentMessageSequenceNumber uint16
}

func newFragmentBuffer() *fragmentBuffer {
	return &fragmentBuffer{cache: map[uint16][]*fragment{}}
}

// Attempts to push a DTLS packet to the fragmentBuffer
// when it returns true it means the fragmentBuffer has inserted and the buffer shouldn't be handled
// when an error returns it is fatal, and the DTLS connection should be stopped
func (f *fragmentBuffer) push(buf []byte) (bool, error) {
	frag := new(fragment)
	if err := frag.recordLayerHeader.Unmarshal(buf); err != nil {
		return false, err
	}

	// fragment isn't a handshake, we don't need to handle it
	if frag.recordLayerHeader.contentType != contentTypeHandshake {
		return false, nil
	}

	if err := frag.handshakeHeader.Unmarshal(buf[recordLayerHeaderSize:]); err != nil {
		return false, err
	}

	if _, ok := f.cache[frag.handshakeHeader.messageSequence]; !ok {
		f.cache[frag.handshakeHeader.messageSequence] = []*fragment{}
	}

	// Discard all headers, when rebuilding the packet we will re-build
	frag.data = append([]byte{}, buf[recordLayerHeaderSize+handshakeHeaderLength:]...)
	f.cache[frag.handshakeHeader.messageSequence] = append(f.cache[frag.handshakeHeader.messageSequence], frag)

	return true, nil
}

func (f *fragmentBuffer) pop() []byte {
	frags, ok := f.cache[f.currentMessageSequenceNumber]
	if !ok {
		return nil
	}

	// Go doesn't support recursive lambdas
	var appendMessage func(targetOffset uint32) bool

	rawMessage := []byte{}
	appendMessage = func(targetOffset uint32) bool {
		for _, f := range frags {
			if f.handshakeHeader.fragmentOffset == targetOffset {
				fragmentEnd := (f.handshakeHeader.fragmentOffset + f.handshakeHeader.fragmentLength)
				if fragmentEnd != f.handshakeHeader.length {
					if !appendMessage(fragmentEnd) {
						return false
					}
				}

				rawMessage = append(f.data, rawMessage...)
				return true
			}
		}
		return false
	}

	// Recursively collect up
	if !appendMessage(0) {
		return nil
	}

	firstHeader := frags[0].handshakeHeader
	firstHeader.fragmentOffset = 0
	firstHeader.fragmentLength = firstHeader.length

	rawHeader, err := firstHeader.Marshal()
	if err != nil {
		return nil
	}

	delete(f.cache, f.currentMessageSequenceNumber)
	f.currentMessageSequenceNumber++
	return append(rawHeader, rawMessage...)
}
