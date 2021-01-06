package interceptor

import (
	"strings"
	"sync/atomic"

	. "github.com/pion/interceptor"
	"github.com/pion/rtp"
)

// ReTime interceptor replaces timestamps of all outgoing video packets.
type ReTime struct {
	NoOp
	timestamp uint32
}

// BindLocalStream modifies any outgoing RTP packets.
func (i *ReTime) BindLocalStream(info *StreamInfo, writer RTPWriter) RTPWriter {
	// use with video packets only
	if strings.HasPrefix(info.MimeType, "video/") {
		return RTPWriterFunc(func(header *rtp.Header, payload []byte, attributes Attributes) (int, error) {
			h := *header
			h.Timestamp = i.GetTimestamp()
			return writer.Write(&h, payload, attributes)
		})
	}
	return writer
}

func (i *ReTime) SetTimestamp(ts uint32) {
	atomic.StoreUint32(&i.timestamp, ts)
}

func (i *ReTime) GetTimestamp() uint32 {
	return atomic.LoadUint32(&i.timestamp)
}
