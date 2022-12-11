package vpx

type Options struct {
	// Target bandwidth to use for this stream, in kilobits per second.
	Bitrate uint
	// Force keyframe interval.
	KeyframeInt uint
}

type Option func(*Options)

func WithOptions(arg Options) Option {
	return func(args *Options) {
		args.Bitrate = arg.Bitrate
		args.KeyframeInt = arg.KeyframeInt
	}
}
