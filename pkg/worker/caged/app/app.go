package app

type App interface {
	AudioSampleRate() int
	Init() error
	ViewportSize() (int, int)
	Start()
	Close()

	SetAudioCb(func(Audio))
	SetVideoCb(func(Video))
	SendControl(port int, data []byte)
}

type Audio struct {
	Data     []int16
	Duration int32 // up to 6y nanosecond-wise
}

type Video struct {
	Frame    RawFrame
	Duration int32
}

type RawFrame struct {
	Data   []byte
	Stride int
	W, H   int
}
