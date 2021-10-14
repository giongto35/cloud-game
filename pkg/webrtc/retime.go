package webrtc

import (
	"strings"
	"sync/atomic"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

// ReTimeInterceptor replaces timestamps of all outgoing video packets.
type ReTimeInterceptor struct {
	interceptor.NoOp
	timestamp *uint32
}

func (i *ReTimeInterceptor) NewInterceptor(_ string) (interceptor.Interceptor, error) { return i, nil }

// BindLocalStream modifies any outgoing RTP packets.
func (i *ReTimeInterceptor) BindLocalStream(info *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
	// use with video packets only
	if !strings.HasPrefix(info.MimeType, "video/") {
		return writer
	}
	return interceptor.RTPWriterFunc(func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
		h := *header
		h.Timestamp = atomic.LoadUint32(i.timestamp)
		return writer.Write(&h, payload, attributes)
	})
}
