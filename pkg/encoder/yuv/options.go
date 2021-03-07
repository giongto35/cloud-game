package yuv

type Options struct {
	ChromaP  ChromaPos
	Threaded bool
	Threads  int
}

func (o *Options) override(options ...Option) {
	for _, opt := range options {
		opt(o)
	}
}

type Option func(*Options)

func Threaded(t bool) Option {
	return func(opts *Options) {
		opts.Threaded = t
	}
}

func Threads(t int) Option {
	return func(opts *Options) {
		opts.Threads = t
	}
}

func ChromaP(cp ChromaPos) Option {
	return func(opts *Options) {
		opts.ChromaP = cp
	}
}

// WithOptions used for config files.
func WithOptions(arg Options) Option {
	return func(args *Options) {
		args.ChromaP = arg.ChromaP
		args.Threaded = arg.Threaded
		args.Threads = arg.Threads
	}
}
