package webrtc

import (
	"strings"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

// CustomTrackSample is used just for adding custom timestamps
// into outgoing packets, since packetizer is not accessible anymore.
// Use webrtc.TrackLocalStaticSample instead if you use constant rate streams.
type CustomTrackSample struct {
	packetizer rtp.Packetizer
	rtpTrack   *webrtc.TrackLocalStaticRTP
	clockRate  float64
	mu         sync.RWMutex
}

func NewCustomTrackSample(c webrtc.RTPCodecCapability, id, streamID string) (*CustomTrackSample, error) {
	rtpTrack, err := webrtc.NewTrackLocalStaticRTP(c, id, streamID)
	if err != nil {
		return nil, err
	}
	return &CustomTrackSample{rtpTrack: rtpTrack}, nil
}

func (s *CustomTrackSample) ID() string { return s.rtpTrack.ID() }

func (s *CustomTrackSample) StreamID() string { return s.rtpTrack.StreamID() }

func (s *CustomTrackSample) Kind() webrtc.RTPCodecType { return s.rtpTrack.Kind() }

func (s *CustomTrackSample) Codec() webrtc.RTPCodecCapability { return s.rtpTrack.Codec() }

func (s *CustomTrackSample) Bind(t webrtc.TrackLocalContext) (webrtc.RTPCodecParameters, error) {
	rtpOutboundMTU := 1200
	codec, err := s.rtpTrack.Bind(t)
	if err != nil {
		return codec, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.packetizer != nil {
		return codec, nil
	}

	payloader, err := payloaderForCodec(codec.RTPCodecCapability)
	if err != nil {
		return codec, err
	}

	s.packetizer = rtp.NewPacketizer(
		rtpOutboundMTU,
		0, // Value is handled when writing
		0, // Value is handled when writing
		payloader,
		rtp.NewRandomSequencer(),
		codec.ClockRate,
	)
	s.clockRate = float64(codec.RTPCodecCapability.ClockRate)
	return codec, nil
}

func (s *CustomTrackSample) Unbind(t webrtc.TrackLocalContext) error {
	return s.rtpTrack.Unbind(t)
}

func (s *CustomTrackSample) WriteSampleWithTimestamp(sample media.Sample, timestamp uint32) (err error) {
	s.mu.RLock()
	p := s.packetizer
	clockRate := s.clockRate
	s.mu.RUnlock()

	if p == nil {
		return nil
	}

	samples := sample.Duration.Seconds() * clockRate
	packets := p.(rtp.Packetizer).Packetize(sample.Data, uint32(samples))
	for _, p := range packets {
		p.Timestamp = timestamp
		err = s.rtpTrack.WriteRTP(p)
	}

	return
}

func payloaderForCodec(codec webrtc.RTPCodecCapability) (rtp.Payloader, error) {
	switch strings.ToLower(codec.MimeType) {
	case strings.ToLower(webrtc.MimeTypeH264):
		return &codecs.H264Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeOpus):
		return &codecs.OpusPayloader{}, nil
	case strings.ToLower(webrtc.MimeTypeVP8):
		return &codecs.VP8Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeVP9):
		return &codecs.VP9Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeG722):
		return &codecs.G722Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypePCMU), strings.ToLower(webrtc.MimeTypePCMA):
		return &codecs.G711Payloader{}, nil
	default:
		return nil, webrtc.ErrNoPayloaderForCodec
	}
}
