package h264

type Options struct {
	// Constant Rate Factor (CRF)
	// This method allows the encoder to attempt to achieve a certain output quality for the whole file
	// when output file size is of less importance.
	// The range of the CRF scale is 0–51, where 0 is lossless, 23 is the default, and 51 is worst quality possible.
	Crf uint8
	// film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency.
	Tune string
	// ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo.
	Preset string
	// baseline, main, high, high10, high422, high444.
	Profile  string
	LogLevel int32
}

type Option func(*Options)

func WithOptions(arg Options) Option {
	return func(args *Options) {
		args.Crf = arg.Crf
		args.Tune = arg.Tune
		args.Preset = arg.Preset
		args.Profile = arg.Profile
		args.LogLevel = arg.LogLevel
	}
}
func Crf(arg uint8) Option      { return func(args *Options) { args.Crf = arg } }
func Tune(arg string) Option    { return func(args *Options) { args.Tune = arg } }
func Preset(arg string) Option  { return func(args *Options) { args.Preset = arg } }
func Profile(arg string) Option { return func(args *Options) { args.Profile = arg } }
func LogLevel(arg int32) Option { return func(args *Options) { args.LogLevel = arg } }
