package encoder

type Encoder struct {
	Audio Audio
	Video Video
}

type Audio struct {
	Frame int
}

type Video struct {
	Codec string
	H264  struct {
		Crf      uint8
		Preset   string
		Profile  string
		Tune     string
		LogLevel int
	}
	Vpx struct {
		Bitrate          uint
		KeyframeInterval uint
	}
}
