package config

type Opus struct {
	Hz      int
	Ch      int
	FrameMs float64
}

func DefaultOpusCfg() Opus {
	return Opus{
		Hz:      48000,
		Ch:      2,
		FrameMs: 20,
	}
}
