// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"fmt"
	"io"
	"unsafe"
)

/*
#cgo pkg-config: opusfile
#include <opusfile.h>
#include <string.h>

extern struct OpusFileCallbacks callbacks;

*/
import "C"

// Stream wraps a io.Reader in a decoding layer. It provides an API similar to
// io.Reader, but it provides raw PCM data instead of the encoded Opus data.
//
// This is not the same as directly decoding the bytes on the io.Reader; opus
// streams are Ogg Opus audio streams, which package raw Opus data.
//
// This wraps libopusfile. For more information, see the api docs on xiph.org:
//
// https://www.opus-codec.org/docs/opusfile_api-0.7/index.html
type Stream struct {
	id      uintptr
	oggfile *C.OggOpusFile
	read    io.Reader
	// Preallocated buffer to pass to the reader
	buf []byte
}

var streams = newStreamsMap()

//export go_readcallback
func go_readcallback(p unsafe.Pointer, cbuf *C.uchar, cmaxbytes C.int) C.int {
	streamId := uintptr(p)
	stream := streams.Get(streamId)
	if stream == nil {
		// This is bad
		return -1
	}

	maxbytes := int(cmaxbytes)
	if maxbytes > cap(stream.buf) {
		maxbytes = cap(stream.buf)
	}
	// Don't bother cleaning up old data because that's not required by the
	// io.Reader API.
	n, err := stream.read.Read(stream.buf[:maxbytes])
	// Go allows returning non-nil error (like EOF) and n>0, libopusfile doesn't
	// expect that. So return n first to indicate the valid bytes, let the
	// subsequent call (which will be n=0, same-error) handle the actual error.
	if n == 0 && err != nil {
		if err == io.EOF {
			return 0
		} else {
			return -1
		}
	}
	C.memcpy(unsafe.Pointer(cbuf), unsafe.Pointer(&stream.buf[0]), C.size_t(n))
	return C.int(n)
}

// NewStream creates and initializes a new stream. Don't call .Init() on this.
func NewStream(read io.Reader) (*Stream, error) {
	var s Stream
	err := s.Init(read)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Init initializes a stream with an io.Reader to fetch opus encoded data from
// on demand. Errors from the reader are all transformed to an EOF, any actual
// error information is lost. The same happens when a read returns succesfully,
// but with zero bytes.
func (s *Stream) Init(read io.Reader) error {
	if s.oggfile != nil {
		return fmt.Errorf("opus stream is already initialized")
	}
	if read == nil {
		return fmt.Errorf("Reader must be non-nil")
	}

	s.read = read
	s.buf = make([]byte, maxEncodedFrameSize)
	s.id = streams.NextId()
	var errno C.int

	// Immediately delete the stream after .Init to avoid leaking if the
	// caller forgets to (/ doesn't want to) call .Close(). No need for that,
	// since the callback is only ever called during a .Read operation; just
	// Save and Delete from the map around that every time a reader function is
	// called.
	streams.Save(s)
	defer streams.Del(s)
	oggfile := C.op_open_callbacks(
		// "C code may not keep a copy of a Go pointer after the call returns."
		unsafe.Pointer(s.id),
		&C.callbacks,
		nil,
		0,
		&errno)
	if errno != 0 {
		return StreamError(errno)
	}
	s.oggfile = oggfile
	return nil
}

// Read a chunk of raw opus data from the stream and decode it. Returns the
// number of decoded samples per channel. This means that a dual channel
// (stereo) feed will have twice as many samples as the value returned.
//
// Read may successfully read less bytes than requested, but it will never read
// exactly zero bytes succesfully if a non-zero buffer is supplied.
//
// The number of channels in the output data must be known in advance. It is
// possible to extract this information from the stream itself, but I'm not
// motivated to do that. Feel free to send a pull request.
func (s *Stream) Read(pcm []int16) (int, error) {
	if s.oggfile == nil {
		return 0, fmt.Errorf("opus stream is uninitialized or already closed")
	}
	if len(pcm) == 0 {
		return 0, nil
	}
	streams.Save(s)
	defer streams.Del(s)
	n := C.op_read(
		s.oggfile,
		(*C.opus_int16)(&pcm[0]),
		C.int(len(pcm)),
		nil)
	if n < 0 {
		return 0, StreamError(n)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}

// ReadFloat32 is the same as Read, but decodes to float32 instead of int16.
func (s *Stream) ReadFloat32(pcm []float32) (int, error) {
	if s.oggfile == nil {
		return 0, fmt.Errorf("opus stream is uninitialized or already closed")
	}
	if len(pcm) == 0 {
		return 0, nil
	}
	streams.Save(s)
	defer streams.Del(s)
	n := C.op_read_float(
		s.oggfile,
		(*C.float)(&pcm[0]),
		C.int(len(pcm)),
		nil)
	if n < 0 {
		return 0, StreamError(n)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}

func (s *Stream) Close() error {
	if s.oggfile == nil {
		return fmt.Errorf("opus stream is uninitialized or already closed")
	}
	C.op_free(s.oggfile)
	if closer, ok := s.read.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
