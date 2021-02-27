package encoder

type Encoder struct {
	Audio       Audio
	Video       Video
	WithoutGame bool
}

type Audio struct {
	Channels  int
	Frame     int
	Frequency int
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

func (a *Audio) GetFrameDuration() int {
	return a.Frequency * a.Frame / 1000 * a.Channels
}
