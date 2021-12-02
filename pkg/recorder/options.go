package recorder

type Options struct {
	Dir                   string
	Fps                   float64
	Frequency             int
	Game                  string
	ImageCompressionLevel int
	Name                  string
	Zip                   bool
}

type Option func(*Options)

type Meta struct {
	UserName string
}
