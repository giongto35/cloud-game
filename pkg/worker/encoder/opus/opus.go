package opus

/*
#cgo pkg-config: opus
#cgo CFLAGS: -Wall -O3

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

type Opus struct {
	mem []byte
	st  *C.struct_OpusEncoder
}

// NewOpusEncoder creates new Opus encoder.
func NewOpusEncoder(sampleRate int, app Application) (*Opus, error) {
	var enc Opus
	if enc.st != nil {
		return nil, fmt.Errorf("opus: encoder reinit")
	}
	const stereo = C.int(2)
	enc.mem = make([]byte, C.opus_encoder_get_size(stereo))
	enc.st = (*C.OpusEncoder)(unsafe.Pointer(&enc.mem[0]))
	err := unwrap(C.opus_encoder_init(enc.st, C.opus_int32(sampleRate), stereo, C.int(app)))
	if err != nil {
		return nil, fmt.Errorf("opus: initializatoin error (%v)", err)
	}
	return &enc, nil
}

// Encode converts raw PCM samples into the supplied Opus buffer.
// Returns the number of bytes converted.
func (enc *Opus) Encode(pcm []int16, data []byte) (rez int, err error) {
	if len(pcm) == 0 {
		return
	}
	n := C.opus_encode(enc.st, (*C.opus_int16)(&pcm[0]), C.int(len(pcm)>>1), (*C.uchar)(&data[0]), C.opus_int32(cap(data)))
	if n > 0 {
		rez = int(n)
	}
	return rez, unwrap(n)
}

// SampleRate returns the sample rate of the encoder.
func (enc *Opus) SampleRate() (int, error) {
	var sampleRate C.opus_int32
	res := C.get_sample_rate(enc.st, &sampleRate)
	return int(sampleRate), unwrap(res)
}

// Bitrate returns the bitrate of the encoder.
func (enc *Opus) Bitrate() (int, error) {
	var bitrate C.opus_int32
	res := C.get_bitrate(enc.st, &bitrate)
	return int(bitrate), unwrap(res)
}

// SetBitrate sets the bitrate of the encoder.
// BitrateMax / BitrateAuto can be used here.
func (enc *Opus) SetBitrate(b Bitrate) error {
	return unwrap(C.set_bitrate(enc.st, C.opus_int32(b)))
}

// Complexity returns the value of the complexity.
func (enc *Opus) Complexity() (int, error) {
	var complexity C.opus_int32
	res := C.get_complexity(enc.st, &complexity)
	return int(complexity), unwrap(res)
}

// SetComplexity sets the complexity factor for the encoder.
// Complexity is a value from 1 to 10, where 1 is the lowest complexity and 10 is the highest.
func (enc *Opus) SetComplexity(complexity int) error {
	return unwrap(C.set_complexity(enc.st, C.opus_int32(complexity)))
}

// DTX says if discontinuous transmission is enabled.
func (enc *Opus) DTX() (bool, error) {
	var dtx C.opus_int32
	res := C.get_dtx(enc.st, &dtx)
	return dtx > 0, unwrap(res)
}

// SetDTX switches discontinuous transmission.
func (enc *Opus) SetDTX(dtx bool) error {
	var i int
	if dtx {
		i = 1
	}
	return unwrap(C.set_dtx(enc.st, C.opus_int32(i)))
}

// MaxBandwidth returns the maximum allowed bandpass value.
func (enc *Opus) MaxBandwidth() (Bandwidth, error) {
	var b C.opus_int32
	res := C.get_max_bandwidth(enc.st, &b)
	return Bandwidth(b), unwrap(res)
}

// SetMaxBandwidth sets the upper limit of the bandpass.
func (enc *Opus) SetMaxBandwidth(b Bandwidth) error {
	return unwrap(C.set_max_bandwidth(enc.st, C.opus_int32(b)))
}

// FEC says if forward error correction (FEC) is enabled.
func (enc *Opus) FEC() (bool, error) {
	var fec C.opus_int32
	res := C.get_inband_fec(enc.st, &fec)
	return fec > 0, unwrap(res)
}

// SetFEC switches the forward error correction (FEC).
func (enc *Opus) SetFEC(fec bool) error {
	var i int
	if fec {
		i = 1
	}
	return unwrap(C.set_inband_fec(enc.st, C.opus_int32(i)))
}

// PacketLossPerc returns configured packet loss percentage.
func (enc *Opus) PacketLossPerc() (int, error) {
	var lossPerc C.opus_int32
	res := C.get_packet_loss_perc(enc.st, &lossPerc)
	return int(lossPerc), unwrap(res)
}

// SetPacketLossPerc sets expected packet loss percentage.
func (enc *Opus) SetPacketLossPerc(lossPerc int) error {
	return unwrap(C.set_packet_loss_perc(enc.st, C.opus_int32(lossPerc)))
}

func (enc *Opus) ResetState() error { return unwrap(C.reset_state(enc.st)) }

func (e Error) Error() string { return fmt.Sprintf("opus: %v", C.GoString(C.opus_strerror(C.int(e)))) }

func unwrap(error C.int) (err error) {
	if error < C.OPUS_OK {
		err = Error(int(error))
	}
	return
}

func CodecVersion() string { return C.GoString(C.opus_get_version_string()) }
