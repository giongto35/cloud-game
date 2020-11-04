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
	p.sampleCount = GetSampleCount(p.e.getSampleRate(), p.e.getChannelCount(), p.e.getFrameSize())
	SamplesPerFrame = p.sampleCount / 2

	return p
}

func (p Processor) Encode(pcm []int16, sampleRate int) []byte {
	data := pcm
	if p.resampling && sampleRate != p.sampleRate {
		data = resample(pcm, p.sampleCount, sampleRate, p.sampleRate)
	}
	return p.e.encode(data)
}

// GetSampleCount returns a number of audio samples for the given frame duration (ms).
func GetSampleCount(sampleRate int, channels int, frameTime float64) int {
	return int(float64(sampleRate) / float64(1000) * frameTime * float64(channels))
}

// resample processes raw PCM (interleaved) samples with a simple linear resampling function.
// Zero samples in the right or left channel are replaced with the previous sample.
//
// Bad:
// 	- Can resample with static noise.
// 	- O(n+n*log(n)).
// 	- Hardcoded for the stereo PCM.
// 	- Not tested for down-sample.
//
// !to check ratio based approach (one-off boundaries rounding error)
func resample(pcm []int16, samples int, srcSR int, dstSR int) []int16 {
	l, r, mux := make([]int16, samples/2), make([]int16, samples/2), make([]int16, samples)

	// split samples to spread inside the new time frame
	for i, n := 0, len(pcm)-1; i < n; i += 2 {
		index := i / 2 * dstSR / srcSR
		l[index], r[index] = pcm[i], pcm[i+1]
	}

	// interpolation (stretch samples)
	for i, n := 1, len(l); i < n; i++ {
		if l[i] == 0 {
			l[i] = l[i-1]
		}
		if r[i] == 0 {
			r[i] = r[i-1]
		}
	}

	// merge l+r channels
	for i := 0; i < samples-1; i += 2 {
		mux[i], mux[i+1] = l[i/2], r[i/2]
	}

	return mux
}
