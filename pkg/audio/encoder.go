package audio

type Encoder interface {
	encode(pcm []int16) []byte

	getSampleRate() int
	getChannelCount() int
	getFrameSize() float64
}

type Processor struct {
	e Encoder

	resampling bool

	// cache
	channels    int
	sampleRate  int
	sampleCount int
}

// hack for WebRTC module
var SamplesPerFrame = 0

func NewAudioProcessor(enc Encoder, err error) Processor {
	if err != nil {
		panic(err)
	}

	p := Processor{
		e:          enc,
		resampling: true,
	}
	p.channels = p.e.getChannelCount()
	p.sampleRate = p.e.getSampleRate()
	p.sampleCount = p.GetSampleCount(p.e.getSampleRate(), p.e.getChannelCount(), p.e.getFrameSize())
	SamplesPerFrame = p.sampleCount / 2

	return p
}

func (p Processor) GetSampleCount(sampleRate int, channels int, frameSize float64) int {
	return int(float64(sampleRate) / float64(1000) * frameSize * float64(channels))
}

func (p Processor) Encode(pcm []int16, sampleRate int) []byte {
	data := pcm
	if p.resampling && sampleRate != p.sampleRate {
		data = p.resample(pcm, p.sampleCount, sampleRate, p.sampleRate)
	}
	return p.e.encode(data)
}

// resample processes raw PCM samples with a simple linear resampling function.
// Zero samples in the right or left channel are replaced with the previous sample.
//
// Bad:
// 	- Can resample with static noise.
// 	- O(n+n*log(n)).
// 	- Hardcoded for stereo PCM samples.
// 	- Not tested for down-sample.
//
// !to check ratio based approach (one-off boundaries rounding error)
func (p Processor) resample(pcm []int16, samples int, srcSR int, dstSR int) []int16 {
	l, r, mux := make([]int16, samples/2), make([]int16, samples/2), make([]int16, samples)

	for i := 0; i < len(pcm)-1; i += 2 {
		index := i / 2 * dstSR / srcSR
		l[index], r[index] = pcm[i], pcm[i+1]
	}

	for i := 1; i < len(l); i++ {
		if l[i] == 0 {
			l[i] = l[i-1]
		}
		if r[i] == 0 {
			r[i] = r[i-1]
		}
	}

	for i := 0; i < samples-1; i += 2 {
		mux[i], mux[i+1] = l[i/2], r[i/2]
	}

	return mux
}
