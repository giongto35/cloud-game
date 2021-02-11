package h264

type Options struct {
	// film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency.
	Tune string
	// ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo.
	Preset string
	// baseline, main, high, high10, high422, high444.
	Profile  string
	LogLevel int32
}

type Option func(*Options)

func Tune(arg string) Option    { return func(args *Options) { args.Tune = arg } }
func Preset(arg string) Option  { return func(args *Options) { args.Preset = arg } }
func Profile(arg string) Option { return func(args *Options) { args.Profile = arg } }
func LogLevel(arg int32) Option { return func(args *Options) { args.LogLevel = arg } }
