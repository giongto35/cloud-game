package opus

/*
#cgo pkg-config: opus
#cgo st LDFLAGS: -l:libopus.a

#include <opus.h>

int get_bitrate(OpusEncoder *st, opus_int32 *bitrate) { return opus_encoder_ctl(st, OPUS_GET_BITRATE(bitrate)); }
int get_complexity(OpusEncoder *st, opus_int32 *complexity) { return opus_encoder_ctl(st, OPUS_GET_COMPLEXITY(complexity)); }
int get_dtx(OpusEncoder *st, opus_int32 *dtx) { return opus_encoder_ctl(st, OPUS_GET_DTX(dtx)); }
int get_inband_fec(OpusEncoder *st, opus_int32 *fec) { return opus_encoder_ctl(st, OPUS_GET_INBAND_FEC(fec)); }
int get_max_bandwidth(OpusEncoder *st, opus_int32 *max_bw) {	return opus_encoder_ctl(st, OPUS_GET_MAX_BANDWIDTH(max_bw)); }
int get_packet_loss_perc(OpusEncoder *st, opus_int32 *loss_perc) { return opus_encoder_ctl(st, OPUS_GET_PACKET_LOSS_PERC(loss_perc)); }
int get_sample_rate(OpusEncoder *st, opus_int32 *sample_rate) { return opus_encoder_ctl(st, OPUS_GET_SAMPLE_RATE(sample_rate)); }
int set_bitrate(OpusEncoder *st, opus_int32 bitrate) { return opus_encoder_ctl(st, OPUS_SET_BITRATE(bitrate)); }
int set_complexity(OpusEncoder *st, opus_int32 complexity) { return opus_encoder_ctl(st, OPUS_SET_COMPLEXITY(complexity)); }
int set_dtx(OpusEncoder *st, opus_int32 use_dtx) { return opus_encoder_ctl(st, OPUS_SET_DTX(use_dtx)); }
int set_inband_fec(OpusEncoder *st, opus_int32 fec) { return opus_encoder_ctl(st, OPUS_SET_INBAND_FEC(fec)); }
int set_max_bandwidth(OpusEncoder *st, opus_int32 max_bw) { return opus_encoder_ctl(st, OPUS_SET_MAX_BANDWIDTH(max_bw)); }
int set_packet_loss_perc(OpusEncoder *st, opus_int32 loss_perc) { return opus_encoder_ctl(st, OPUS_SET_PACKET_LOSS_PERC(loss_perc)); }
int reset_state(OpusEncoder *st) { return opus_encoder_ctl(st, OPUS_RESET_STATE); }
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type (
	Application int
	Bandwidth   int
	Bitrate     int
	Error       int
)

const (
	// AppRestrictedLowDelay optimizes encoding for low latency applications
	AppRestrictedLowDelay = Application(C.OPUS_APPLICATION_RESTRICTED_LOWDELAY)
	// FullBand is 20 kHz bandpass
	FullBand = Bandwidth(C.OPUS_BANDWIDTH_FULLBAND)
)
const stereo = C.int(2)

type Encoder struct {
	mem []byte
	out []byte
	st  *C.struct_OpusEncoder
}

func NewEncoder(outFq int) (*Encoder, error) {
	mem := make([]byte, C.opus_encoder_get_size(stereo))
	out := make([]byte, 1024)
	enc := Encoder{
		mem: mem,
		st:  (*C.OpusEncoder)(unsafe.Pointer(&mem[0])),
		out: out,
	}
	err := unwrap(C.opus_encoder_init(enc.st, C.opus_int32(outFq), stereo, C.int(AppRestrictedLowDelay)))
	if err != nil {
		return nil, fmt.Errorf("opus: initializatoin error (%v)", err)
	}
	_ = enc.SetMaxBandwidth(FullBand)
	_ = enc.SetBitrate(96000)
	_ = enc.SetComplexity(5)

	return &enc, nil
}

func (e *Encoder) Reset() error { return e.ResetState() }

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, nil
	}
	n := C.opus_encode(e.st, (*C.opus_int16)(&pcm[0]), C.int(len(pcm)>>1), (*C.uchar)(&e.out[0]), C.opus_int32(cap(pcm)))
	err := unwrap(n)
	// n = 1 is DTX
	if err != nil || n == 1 {
		return []byte{}, err
	}
	return e.out[:int(n)], nil
}

func (e *Encoder) GetInfo() string {
	bitrate, _ := e.Bitrate()
	complexity, _ := e.Complexity()
	dtx, _ := e.DTX()
	fec, _ := e.FEC()
	maxBandwidth, _ := e.MaxBandwidth()
	lossPercent, _ := e.PacketLossPerc()
	sampleRate, _ := e.SampleRate()
	return fmt.Sprintf(
		"%v, Bitrate: %v bps, Complexity: %v, DTX: %v, FEC: %v, Max bandwidth: *%v, Loss%%: %v, Rate: %v Hz",
		CodecVersion(), bitrate, complexity, dtx, fec, maxBandwidth, lossPercent, sampleRate,
	)
}

// SampleRate returns the sample rate of the encoder.
func (e *Encoder) SampleRate() (int, error) {
	var sampleRate C.opus_int32
	res := C.get_sample_rate(e.st, &sampleRate)
	return int(sampleRate), unwrap(res)
}

// Bitrate returns the bitrate of the encoder.
func (e *Encoder) Bitrate() (int, error) {
	var bitrate C.opus_int32
	res := C.get_bitrate(e.st, &bitrate)
	return int(bitrate), unwrap(res)
}

// SetBitrate sets the bitrate of the encoder.
// BitrateMax / BitrateAuto can be used here.
func (e *Encoder) SetBitrate(b Bitrate) error {
	return unwrap(C.set_bitrate(e.st, C.opus_int32(b)))
}

// Complexity returns the value of the complexity.
func (e *Encoder) Complexity() (int, error) {
	var complexity C.opus_int32
	res := C.get_complexity(e.st, &complexity)
	return int(complexity), unwrap(res)
}

// SetComplexity sets the complexity factor for the encoder.
// Complexity is a value from 1 to 10, where 1 is the lowest complexity and 10 is the highest.
func (e *Encoder) SetComplexity(complexity int) error {
	return unwrap(C.set_complexity(e.st, C.opus_int32(complexity)))
}

// DTX says if discontinuous transmission is enabled.
func (e *Encoder) DTX() (bool, error) {
	var dtx C.opus_int32
	res := C.get_dtx(e.st, &dtx)
	return dtx > 0, unwrap(res)
}

// SetDTX switches discontinuous transmission.
func (e *Encoder) SetDTX(dtx bool) error {
	var i int
	if dtx {
		i = 1
	}
	return unwrap(C.set_dtx(e.st, C.opus_int32(i)))
}

// MaxBandwidth returns the maximum allowed bandpass value.
func (e *Encoder) MaxBandwidth() (Bandwidth, error) {
	var b C.opus_int32
	res := C.get_max_bandwidth(e.st, &b)
	return Bandwidth(b), unwrap(res)
}

// SetMaxBandwidth sets the upper limit of the bandpass.
func (e *Encoder) SetMaxBandwidth(b Bandwidth) error {
	return unwrap(C.set_max_bandwidth(e.st, C.opus_int32(b)))
}

// FEC says if forward error correction (FEC) is enabled.
func (e *Encoder) FEC() (bool, error) {
	var fec C.opus_int32
	res := C.get_inband_fec(e.st, &fec)
	return fec > 0, unwrap(res)
}

// SetFEC switches the forward error correction (FEC).
func (e *Encoder) SetFEC(fec bool) error {
	var i int
	if fec {
		i = 1
	}
	return unwrap(C.set_inband_fec(e.st, C.opus_int32(i)))
}

// PacketLossPerc returns configured packet loss percentage.
func (e *Encoder) PacketLossPerc() (int, error) {
	var lossPerc C.opus_int32
	res := C.get_packet_loss_perc(e.st, &lossPerc)
	return int(lossPerc), unwrap(res)
}

// SetPacketLossPerc sets expected packet loss percentage.
func (e *Encoder) SetPacketLossPerc(lossPerc int) error {
	return unwrap(C.set_packet_loss_perc(e.st, C.opus_int32(lossPerc)))
}

func (e *Encoder) ResetState() error { return unwrap(C.reset_state(e.st)) }

func (e Error) Error() string { return fmt.Sprintf("opus: %v", C.GoString(C.opus_strerror(C.int(e)))) }

func unwrap(error C.int) (err error) {
	if error < C.OPUS_OK {
		err = Error(int(error))
	}
	return
}

func CodecVersion() string { return C.GoString(C.opus_get_version_string()) }
