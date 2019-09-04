// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"fmt"
)

/*
#cgo pkg-config: opus opusfile
#include <opus.h>
#include <opusfile.h>
*/
import "C"

type Error int

var _ error = Error(0)

// Libopus errors
const (
	ErrOK             = Error(C.OPUS_OK)
	ErrBadArg         = Error(C.OPUS_BAD_ARG)
	ErrBufferTooSmall = Error(C.OPUS_BUFFER_TOO_SMALL)
	ErrInternalError  = Error(C.OPUS_INTERNAL_ERROR)
	ErrInvalidPacket  = Error(C.OPUS_INVALID_PACKET)
	ErrUnimplemented  = Error(C.OPUS_UNIMPLEMENTED)
	ErrInvalidState   = Error(C.OPUS_INVALID_STATE)
	ErrAllocFail      = Error(C.OPUS_ALLOC_FAIL)
)

// Error string (in human readable format) for libopus errors.
func (e Error) Error() string {
	return fmt.Sprintf("opus: %s", C.GoString(C.opus_strerror(C.int(e))))
}

type StreamError int

var _ error = StreamError(0)

// Libopusfile errors. The names are copied verbatim from the libopusfile
// library.
const (
	ErrStreamFalse        = StreamError(C.OP_FALSE)
	ErrStreamEOF          = StreamError(C.OP_EOF)
	ErrStreamHole         = StreamError(C.OP_HOLE)
	ErrStreamRead         = StreamError(C.OP_EREAD)
	ErrStreamFault        = StreamError(C.OP_EFAULT)
	ErrStreamImpl         = StreamError(C.OP_EIMPL)
	ErrStreamInval        = StreamError(C.OP_EINVAL)
	ErrStreamNotFormat    = StreamError(C.OP_ENOTFORMAT)
	ErrStreamBadHeader    = StreamError(C.OP_EBADHEADER)
	ErrStreamVersion      = StreamError(C.OP_EVERSION)
	ErrStreamNotAudio     = StreamError(C.OP_ENOTAUDIO)
	ErrStreamBadPacked    = StreamError(C.OP_EBADPACKET)
	ErrStreamBadLink      = StreamError(C.OP_EBADLINK)
	ErrStreamNoSeek       = StreamError(C.OP_ENOSEEK)
	ErrStreamBadTimestamp = StreamError(C.OP_EBADTIMESTAMP)
)

func (i StreamError) Error() string {
	switch i {
	case ErrStreamFalse:
		return "OP_FALSE"
	case ErrStreamEOF:
		return "OP_EOF"
	case ErrStreamHole:
		return "OP_HOLE"
	case ErrStreamRead:
		return "OP_EREAD"
	case ErrStreamFault:
		return "OP_EFAULT"
	case ErrStreamImpl:
		return "OP_EIMPL"
	case ErrStreamInval:
		return "OP_EINVAL"
	case ErrStreamNotFormat:
		return "OP_ENOTFORMAT"
	case ErrStreamBadHeader:
		return "OP_EBADHEADER"
	case ErrStreamVersion:
		return "OP_EVERSION"
	case ErrStreamNotAudio:
		return "OP_ENOTAUDIO"
	case ErrStreamBadPacked:
		return "OP_EBADPACKET"
	case ErrStreamBadLink:
		return "OP_EBADLINK"
	case ErrStreamNoSeek:
		return "OP_ENOSEEK"
	case ErrStreamBadTimestamp:
		return "OP_EBADTIMESTAMP"
	default:
		return "libopusfile error: %d (unknown code)"
	}
}
