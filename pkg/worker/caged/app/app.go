package app

type App interface {
	AudioSampleRate() int
	AspectRatio() float32
	AspectEnabled() bool
	Init() error
	ViewportSize() (int, int)
	Scale() float64
	Start()
	Close()

	SetAudioCb(func(Audio))
	SetVideoCb(func(Video))
	SetDataCb(func([]byte))
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
