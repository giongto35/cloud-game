package recorder

type Options struct {
	Dir       string
	Fps       float64
	W         int
	H         int
	Stride    int
	Flip      bool
	Frequency int
	Pix       string
	Game      string
	Name      string
	Zip       bool
	Vsync     bool
}

type Meta struct {
	UserName string
}
