package recorder

type Options struct {
	Dir                   string
	Fps                   float64
	Frequency             int
	Game                  string
	ImageCompressionLevel int
	Name                  string
	Zip                   bool
	Vsync                 bool
}

type Meta struct {
	UserName string
}
