package encoder

type VideoCodec int

const (
	H264 VideoCodec = iota
	VPX
)

func (v VideoCodec) String() string {
	if v == H264 {
		return "H264"
	} else {
		return "VPX"
	}
}
