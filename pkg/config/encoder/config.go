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

func (a *Audio) GetFrameSize() int          { return a.GetFrameSizeFor(a.Frequency) }
func (a *Audio) GetFrameSizeFor(hz int) int { return hz * a.Frame / 1000 * a.Channels }
