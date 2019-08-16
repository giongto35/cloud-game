// +build gofuzz

package dtls

import "fmt"

func partialHeaderMismatch(a, b recordLayerHeader) bool {
	// Ignoring content length for now.
	a.contentLen = b.contentLen
	return a != b
}

func FuzzRecordLayer(data []byte) int {
	var r recordLayer
	if err := r.Unmarshal(data); err != nil {
		return 0
	}
	buf, err := r.Marshal()
	if err != nil {
		return 1
	}
	if len(buf) == 0 {
		panic("zero buff")
	}
	var nr recordLayer
	if err = nr.Unmarshal(data); err != nil {
		panic(err)
	}
	if partialHeaderMismatch(nr.recordLayerHeader, r.recordLayerHeader) {
		panic(fmt.Sprintf("header mismatch: %+v != %+v",
			nr.recordLayerHeader, r.recordLayerHeader,
		))
	}

	return 1
}
