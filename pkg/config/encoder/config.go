package encoder

type Encoder struct {
	Audio       Audio
	WithoutGame bool
}

type Audio struct {
	Channels  int
	Frame     int
	Frequency int
}

func (a *Audio) GetFrameDuration() int {
	return a.Frequency * a.Frame / 1000 * a.Channels
}
