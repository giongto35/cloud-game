package opus

/*
#cgo pkg-config: opus

#include <opus.h>

int bridge_encoder_get_bitrate(OpusEncoder *st, opus_int32 *bitrate) { return opus_encoder_ctl(st, OPUS_GET_BITRATE(bitrate)); }
int bridge_encoder_get_complexity(OpusEncoder *st, opus_int32 *complexity) { return opus_encoder_ctl(st, OPUS_GET_COMPLEXITY(complexity)); }
int bridge_encoder_get_dtx(OpusEncoder *st, opus_int32 *dtx) { return opus_encoder_ctl(st, OPUS_GET_DTX(dtx)); }
int bridge_encoder_get_inband_fec(OpusEncoder *st, opus_int32 *fec) { return opus_encoder_ctl(st, OPUS_GET_INBAND_FEC(fec)); }
int bridge_encoder_get_max_bandwidth(OpusEncoder *st, opus_int32 *max_bw) {	return opus_encoder_ctl(st, OPUS_GET_MAX_BANDWIDTH(max_bw)); }
int bridge_encoder_get_packet_loss_perc(OpusEncoder *st, opus_int32 *loss_perc) { return opus_encoder_ctl(st, OPUS_GET_PACKET_LOSS_PERC(loss_perc)); }
int bridge_encoder_get_sample_rate(OpusEncoder *st, opus_int32 *sample_rate) { return opus_encoder_ctl(st, OPUS_GET_SAMPLE_RATE(sample_rate)); }
int bridge_encoder_set_bitrate(OpusEncoder *st, opus_int32 bitrate) { return opus_encoder_ctl(st, OPUS_SET_BITRATE(bitrate)); }
int bridge_encoder_set_complexity(OpusEncoder *st, opus_int32 complexity) { return opus_encoder_ctl(st, OPUS_SET_COMPLEXITY(complexity)); }
int bridge_encoder_set_dtx(OpusEncoder *st, opus_int32 use_dtx) { return opus_encoder_ctl(st, OPUS_SET_DTX(use_dtx)); }
int bridge_encoder_set_inband_fec(OpusEncoder *st, opus_int32 fec) { return opus_encoder_ctl(st, OPUS_SET_INBAND_FEC(fec)); }
int bridge_encoder_set_max_bandwidth(OpusEncoder *st, opus_int32 max_bw) { return opus_encoder_ctl(st, OPUS_SET_MAX_BANDWIDTH(max_bw)); }
int bridge_encoder_set_packet_loss_perc(OpusEncoder *st, opus_int32 loss_perc) { return opus_encoder_ctl(st, OPUS_SET_PACKET_LOSS_PERC(loss_perc)); }
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
	// Optimize encoding for VoIP
	AppVoIP = Application(C.OPUS_APPLICATION_VOIP)
	// Optimize encoding for non-voice signals like music
	AppAudio = Application(C.OPUS_APPLICATION_AUDIO)
	// Optimize encoding for low latency applications
	AppRestrictedLowdelay = Application(C.OPUS_APPLICATION_RESTRICTED_LOWDELAY)

	// Auto/default setting
	BitrateAuto = Bitrate(-1000)
	BitrateMax  = Bitrate(-1)

	// 20 kHz bandpass
	FullBand = Bandwidth(C.OPUS_BANDWIDTH_FULLBAND)
)

const (
	ErrorOK             = Error(C.OPUS_OK)
	ErrorBadArg         = Error(C.OPUS_BAD_ARG)
	ErrorBufferTooSmall = Error(C.OPUS_BUFFER_TOO_SMALL)
	ErrorInternalError  = Error(C.OPUS_INTERNAL_ERROR)
	ErrorInvalidPacket  = Error(C.OPUS_INVALID_PACKET)
	ErrorUnimplemented  = Error(C.OPUS_UNIMPLEMENTED)
	ErrorInvalidState   = Error(C.OPUS_INVALID_STATE)
	ErrorAllocFail      = Error(C.OPUS_ALLOC_FAIL)
)

type LibOpusEncoder struct {
	buf      []byte
	channels int
	ptr      *C.struct_OpusEncoder
}

// NewOpusEncoder creates new Opus encoder.
func NewOpusEncoder(sampleRate int, channels int, app Application) (*LibOpusEncoder, error) {
	var enc LibOpusEncoder
	if enc.ptr != nil {
		return nil, fmt.Errorf("opus: encoder reinit")
	}
	enc.channels = channels
	// !to check mem leak
	enc.buf = make([]byte, C.opus_encoder_get_size(C.int(channels)))
	enc.ptr = (*C.OpusEncoder)(unsafe.Pointer(&enc.buf[0]))
	err := unwrap(C.opus_encoder_init(enc.ptr, C.opus_int32(sampleRate), C.int(channels), C.int(app)))
	if err != nil {
		return nil, fmt.Errorf("opus: initializatoin error (%v)", err)
	}
	return &enc, nil
}

// Encode converts raw PCM samples into the supplied Opus buffer.
// Returns the number of bytes converted.
func (enc *LibOpusEncoder) Encode(pcm []int16, data []byte) (rez int, err error) {
	if len(pcm) == 0 {
		return
	}
	samples := C.int(len(pcm) / enc.channels)
	n := C.opus_encode(enc.ptr, (*C.opus_int16)(&pcm[0]), samples, (*C.uchar)(&data[0]), C.opus_int32(cap(data)))
	if n > 0 {
		rez = int(n)
	}
	return rez, unwrap(n)
}

// SampleRate returns the sample rate of the encoder.
func (enc *LibOpusEncoder) SampleRate() (int, error) {
	var sampleRate C.opus_int32
	res := C.bridge_encoder_get_sample_rate(enc.ptr, &sampleRate)
	return int(sampleRate), unwrap(res)
}

// Bitrate returns the bitrate of the encoder.
func (enc *LibOpusEncoder) Bitrate() (int, error) {
	var bitrate C.opus_int32
	res := C.bridge_encoder_get_bitrate(enc.ptr, &bitrate)
	return int(bitrate), unwrap(res)
}

// SetBitrate sets the bitrate of the encoder.
// BitrateMax / BitrateAuto can be used here.
func (enc *LibOpusEncoder) SetBitrate(b Bitrate) error {
	return unwrap(C.bridge_encoder_set_bitrate(enc.ptr, C.opus_int32(b)))
}

// Complexity returns the value of the complexity.
func (enc *LibOpusEncoder) Complexity() (int, error) {
	var complexity C.opus_int32
	res := C.bridge_encoder_get_complexity(enc.ptr, &complexity)
	return int(complexity), unwrap(res)
}

// SetComplexity sets the complexity factor for the encoder.
// Complexity is a value from 1 to 10, where 1 is the lowest complexity and 10 is the highest.
func (enc *LibOpusEncoder) SetComplexity(complexity int) error {
	return unwrap(C.bridge_encoder_set_complexity(enc.ptr, C.opus_int32(complexity)))
}

// DTX says if discontinuous transmission is enabled.
func (enc *LibOpusEncoder) DTX() (bool, error) {
	var dtx C.opus_int32
	res := C.bridge_encoder_get_dtx(enc.ptr, &dtx)
	return dtx > 0, unwrap(res)
}

// SetDTX switches discontinuous transmission.
func (enc *LibOpusEncoder) SetDTX(dtx bool) error {
	var i int
	if dtx {
		i = 1
	}
	return unwrap(C.bridge_encoder_set_dtx(enc.ptr, C.opus_int32(i)))
}

// MaxBandwidth returns the maximum allowed bandpass value.
func (enc *LibOpusEncoder) MaxBandwidth() (Bandwidth, error) {
	var b C.opus_int32
	res := C.bridge_encoder_get_max_bandwidth(enc.ptr, &b)
	return Bandwidth(b), unwrap(res)
}

// SetMaxBandwidth sets the upper limit of the bandpass.
func (enc *LibOpusEncoder) SetMaxBandwidth(b Bandwidth) error {
	return unwrap(C.bridge_encoder_set_max_bandwidth(enc.ptr, C.opus_int32(b)))
}

// FEC says if forward error correction (FEC) is enabled.
func (enc *LibOpusEncoder) FEC() (bool, error) {
	var fec C.opus_int32
	res := C.bridge_encoder_get_inband_fec(enc.ptr, &fec)
	return fec > 0, unwrap(res)
}

// SetFEC switches the forward error correction (FEC).
func (enc *LibOpusEncoder) SetFEC(fec bool) error {
	var i int
	if fec {
		i = 1
	}
	return unwrap(C.bridge_encoder_set_inband_fec(enc.ptr, C.opus_int32(i)))
}

// PacketLossPerc returns configured packet loss percentage.
func (enc *LibOpusEncoder) PacketLossPerc() (int, error) {
	var lossPerc C.opus_int32
	res := C.bridge_encoder_get_packet_loss_perc(enc.ptr, &lossPerc)
	return int(lossPerc), unwrap(res)
}

// SetPacketLossPerc sets expected packet loss percentage.
func (enc *LibOpusEncoder) SetPacketLossPerc(lossPerc int) error {
	return unwrap(C.bridge_encoder_set_packet_loss_perc(enc.ptr, C.opus_int32(lossPerc)))
}

func (e Error) Error() string {
	return fmt.Sprintf("opus: %v", C.GoString(C.opus_strerror(C.int(e))))
}

func unwrap(error C.int) (err error) {
	if error < C.OPUS_OK {
		err = Error(int(error))
	}
	return
}

func CodecVersion() string { return C.GoString(C.opus_get_version_string()) }
