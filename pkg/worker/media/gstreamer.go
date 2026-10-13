package media

import (
	"cmp"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/klauspost/cpuid/v2"
)

/*
#cgo pkg-config: gstreamer-video-1.0 gstreamer-app-1.0
#include <gst/gst.h>
#include <gst/video/gstvideometa.h>
#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>

static inline GstVideoFormat gstVideoFormatFromString(const gchar *s) {
	return gst_video_format_from_string(s);
}

static inline void doubleToFraction(const gdouble src, gint *num, gint *den) {
	gst_util_double_to_fraction(src, num, den);
}

// pushVideoBuf wraps frame memory directly.
static inline void pushVideoBuf(GstAppSrc *src, void *data, gsize len,
                                GstVideoFormat fmt, guint w, guint h, gint stride) {
	GstBuffer *buf = gst_buffer_new_wrapped_full(
		GST_MEMORY_FLAG_READONLY, data, len, 0, len, NULL, NULL);
	if (!buf) return;
	if (stride != 0) {
		gsize offset[GST_VIDEO_MAX_PLANES] = {0};
		gint strides[GST_VIDEO_MAX_PLANES] = {stride};
		gst_buffer_add_video_meta_full(buf, GST_VIDEO_FRAME_FLAG_NONE, fmt, w, h, 1, offset, strides);
	}
	gst_app_src_push_buffer(src, buf);
}

// pushAudioBuf copies audio data into a new buffer.
static inline void pushAudioBuf(GstAppSrc *src, void *data, gsize len) {
	GstBuffer *buf = gst_buffer_new_allocate(NULL, len, NULL);
	if (!buf) return;
	gst_buffer_fill(buf, 0, data, len);
	gst_app_src_push_buffer(src, buf);
}

// pullMappedBuffer pulls a sample from the appsink, gets its buffer,
// and maps it for reading. Returns the buffer (with ref held) on success,
// or NULL if no sample/buffer is available.
// The caller must call unmapAndUnref to release.
static inline GstBuffer* pullMappedBuffer(GstAppSink *sink, GstMapInfo *mapInfo) {
	GstSample *sample = gst_app_sink_pull_sample(sink);
	if (!sample) return NULL;
	GstBuffer *buffer = gst_sample_get_buffer(sample);
	if (buffer) gst_buffer_ref(buffer);
	gst_sample_unref(sample);
	if (!buffer) return NULL;
	if (!gst_buffer_map(buffer, mapInfo, GST_MAP_READ)) {
		gst_buffer_unref(buffer);
		return NULL;
	}
	return buffer;
}

static inline void unmapAndUnref(GstBuffer *buffer, GstMapInfo *mapInfo) {
	gst_buffer_unmap(buffer, mapInfo);
	gst_buffer_unref(buffer);
}
*/
import "C"

const (
	pixFmtBGRx  uint32 = 0
	pixFmtBGRA  uint32 = 1
	pixFmtRGB16 uint32 = 2

	defaultAudioFrameMs = 20 * time.Millisecond

	maxForcedKeyframes = 3
)

var forceKeyframeEvent *gst.Event
var cachedSegment *gst.Segment
var cpuCores int
var pixFmtToGst = map[uint32]string{
	pixFmtBGRx:  "BGRx",
	pixFmtBGRA:  "BGRA",
	pixFmtRGB16: "RGB16",
}
var audioBufPool = sync.Pool{New: func() any { b := make([]byte, 4096); return &b }}
var pixFmtCache = map[string]uint32{}

func init() {
	gst.Init(nil)

	s := gst.NewStructure("GstForceKeyUnit")
	s.SetValue("all-headers", true)
	forceKeyframeEvent = gst.NewCustomEvent(gst.EventTypeCustomDownstream, s)

	cpuCores = cmp.Or(cpuid.CPU.PhysicalCores, 4) - 1

	cachedSegment = gst.NewSegment()
	cachedSegment.Init(gst.FormatTime)
}

// GstMediaPipe a video and audio pipline based on GStreamer.
// Very unsafe.
//
// Video encoding is done in a single goroutine to avoid races.
// Audio is pulled from the appsink on GStreamer's own audio thread.
//
// Goroutines (4):
//   - video worker x1 (push+pull loop for video encoding)
//   - audio push x1 (audio queue for GStreamer)
//   - bus messages  x2 (one per pipeline, bus message logging)
type GstMediaPipe struct {
	a, v *pipe

	conf config.Encoder

	pixFmt uint32

	VideoW, VideoH int
	VideoScale     float64
	ScaleMethod    string
	MaxThreads     int
	AudioSrcHz     int
	VideoVFR       bool
	VideoFPS       float64

	oldPf  uint32
	oldRot uint
	bpp    int
	vidFmt uint32 // cached GstVideoFormat enum

	frameI   int  // for forced keyframes
	keyI     int  // periodic keyframe counter
	kfi      int  // 0=GStreamer auto, >0=force keyframe every N frames
	aSegSent bool // for Opusenc bug

	audioCh chan []byte
	onAudio func([]byte, time.Duration)

	// used for reinit
	videoCh   chan videoJob
	videoDone chan struct{}
	reinit    atomic.Bool
	mu        sync.Mutex

	log *logger.Logger
}

type videoJob struct {
	data         []byte
	w, h, stride int
	dur          time.Duration
	cb           func([]byte, time.Duration)
}

type Video struct {
	Frame    RawFrame
	Duration time.Duration
}

type RawFrame struct {
	Data   []byte
	Stride int
	W, H   int
}

type pipe struct {
	pipeline *gst.Pipeline
	source   *app.Source
	sink     *app.Sink
	srcPad   *gst.Pad
	frameDur time.Duration
	stale    atomic.Bool
}

func (p *pipe) src() *C.GstAppSrc      { return (*C.GstAppSrc)(unsafe.Pointer(p.source.Instance())) }
func (p *pipe) sinkPtr() *C.GstAppSink { return (*C.GstAppSink)(unsafe.Pointer(p.sink.Instance())) }
func (p *pipe) stop() {
	if p != nil && p.pipeline != nil {
		p.pipeline.SetState(gst.StateNull)
	}
}

func (p *pipe) destroy() {
	if p == nil {
		return
	}
	if p.pipeline != nil {
		p.pipeline.GetPipelineBus().Post(gst.NewEOSMessage(p.pipeline))
		p.pipeline.SetState(gst.StateNull)
		p.pipeline = nil
	}
	p.sink = nil
	p.source = nil
}

type builder struct{ strings.Builder }

func (b *builder) write(s string)               { b.Builder.WriteString(s) }
func (b *builder) writef(s string, args ...any) { fmt.Fprintf(&b.Builder, s, args...) }
func (b *builder) writeIfSet(s string) {
	if s != "" {
		b.write(s)
	}
}

func NewGstreamer(conf config.Encoder, log *logger.Logger) *GstMediaPipe {
	return &GstMediaPipe{conf: conf, bpp: 4, vidFmt: gstVideoFormat("BGRx"), log: log}
}

func (g *GstMediaPipe) Init() error {
	if err := g.initVideo(); err != nil {
		return fmt.Errorf("gst video init: %w", err)
	}
	if err := g.initAudio(); err != nil {
		return fmt.Errorf("gst audio init: %w", err)
	}
	return nil
}

func (g *GstMediaPipe) initAudio() (err error) {
	srcHz := g.AudioSrcHz
	g.a, err = buildAudioPipeline(srcHz, g.conf, g.log)
	if err != nil {
		return
	}
	g.a.sink.SetCallbacks(&app.SinkCallbacks{NewSampleFunc: g.pullAudio})
	g.audioCh = make(chan []byte, 2)
	go g.pushAudio()
	return g.a.pipeline.SetState(gst.StatePlaying)
}

func (g *GstMediaPipe) initVideo() (err error) {
	w, h, scale := g.VideoW, g.VideoH, g.VideoScale
	sw, sh := round(w, scale), round(h, scale)
	if g.oldRot%180 != 0 {
		w, h = h, w
	}
	g.v, err = buildVideoPipeline(w, h, sw, sh, g.pixFmt, g.ScaleMethod, g.MaxThreads, g.conf,
		g.log, g.VideoFPS, g.VideoVFR)
	if err != nil {
		return
	}

	// if params contain keyframe-mode=disabled, we force periodic keyframes
	// ourselves instead of relying on GStreamer's auto mode
	if opts, _ := g.conf.VideoSettings(); opts != nil && strings.Contains(opts.Params, "keyframe-mode=disabled") {
		g.kfi = cmp.Or(g.conf.Video.KeyframeInterval, 120)
	} else {
		g.kfi = 0
	}

	p := g.v
	fmt := g.vidFmt
	g.videoCh = make(chan videoJob, 1)
	g.videoDone = make(chan struct{})
	go g.videoWorker(p, fmt, g.videoCh, g.videoDone)

	return p.pipeline.SetState(gst.StatePlaying)
}

func (g *GstMediaPipe) SetPixFmt(f uint32) {
	g.oldPf, g.pixFmt = f, f
	g.vidFmt = gstVideoFormat(cmp.Or(pixFmtToGst[f], "RGBA"))
	if f == pixFmtRGB16 {
		g.bpp = 2
	} else {
		g.bpp = 4
	}
}
func (g *GstMediaPipe) SetRot(r uint) { g.oldRot = r }

func (g *GstMediaPipe) Destroy() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.reinit.Store(true)

	if g.audioCh != nil {
		close(g.audioCh)
	}
	if g.videoCh != nil {
		g.v.stop()
		close(g.videoCh)
		<-g.videoDone
	}

	g.v.destroy()
	g.a.destroy()
	g.aSegSent = false
	g.keyI = 0
}

func (g *GstMediaPipe) ProcessAudio(audio []byte, cb func([]byte, time.Duration)) {
	g.onAudio = cb
	// some cores (PS1 while changing video mode) deliver empty audio batches
	if len(audio) == 0 {
		return
	}
	buf := audioBufPool.Get().(*[]byte)
	if cap(*buf) < len(audio) {
		*buf = make([]byte, len(audio))
	}
	*buf = (*buf)[:len(audio)]
	copy(*buf, audio)
	select {
	case g.audioCh <- *buf:
	default:
		audioBufPool.Put(buf)
	}
}

func (g *GstMediaPipe) pushAudio() {
	for data := range g.audioCh {
		if len(data) == 0 {
			continue // &data[0] below would panic
		}
		if g.reinit.Load() || g.a == nil {
			audioBufPool.Put(&data)
			continue
		}
		if !g.aSegSent {
			g.aSegSent = true
			g.a.srcPad.PushEvent(gst.NewSegmentEvent(cachedSegment))
		}
		C.pushAudioBuf(g.a.src(), unsafe.Pointer(&data[0]), C.gsize(len(data)))
		audioBufPool.Put(&data)
	}
}

// pullAudio pulls audio buffers from the appsink when they are available.
func (g *GstMediaPipe) pullAudio(_ *app.Sink) gst.FlowReturn {
	var mapInfo C.GstMapInfo
	buf := C.pullMappedBuffer(g.a.sinkPtr(), &mapInfo)
	if buf == nil {
		return gst.FlowEOS
	}
	defer C.unmapAndUnref(buf, &mapInfo)
	if g.onAudio != nil {
		g.onAudio(unsafe.Slice((*byte)(unsafe.Pointer(mapInfo.data)), int(mapInfo.size)), g.a.frameDur)
	}
	return gst.FlowOK
}

// videoWorker is the single goroutine that does push+pull for video.
func (g *GstMediaPipe) videoWorker(v *pipe, vidFmt uint32, ch <-chan videoJob, done chan<- struct{}) {
	defer close(done)
	for job := range ch {
		if v.stale.Load() || len(job.data) == 0 {
			continue
		}
		C.pushVideoBuf(v.src(),
			unsafe.Pointer(&job.data[0]), C.gsize(len(job.data)),
			C.GstVideoFormat(vidFmt), C.guint(job.w), C.guint(job.h), C.gint(job.stride))

		var mi C.GstMapInfo
		buf := C.pullMappedBuffer(v.sinkPtr(), &mi)
		if buf == nil {
			continue
		}
		data := unsafe.Slice((*byte)(unsafe.Pointer(mi.data)), int(mi.size))
		job.cb(data, job.dur)
		C.unmapAndUnref(buf, &mi)
	}
}

func (g *GstMediaPipe) ProcessVideo(v Video, cb func([]byte, time.Duration)) {
	if g.reinit.Load() {
		return
	}

	// when someone starts watching, force the first few keyframes to avoid stalling
	// and if scene-change detection is disabled (kfi > 0)
	if g.frameI < maxForcedKeyframes || (g.kfi > 0 && g.keyI%g.kfi == 0) {
		g.v.srcPad.PushEvent(forceKeyframeEvent)
		if g.frameI < maxForcedKeyframes {
			g.frameI++
		}
	}
	g.keyI++

	w, h := v.Frame.W, v.Frame.H
	stride := v.Frame.Stride

	data := v.Frame.Data[:stride*h]
	if stride == w*g.bpp {
		stride = 0
	}

	select {
	case g.videoCh <- videoJob{data: data, w: w, h: h, stride: stride, dur: v.Duration, cb: cb}:
	default:
		// if busy the frame is dropped
		// or maybe handle it with gst?
	}
}

func (g *GstMediaPipe) Reinit() error {
	// prevent concurrent reinits
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.v == nil {
		return nil
	}

	g.reinit.Store(true)
	defer g.reinit.Store(false)

	// drop stale frames (for the new pipe)
	g.v.stale.Store(true)

	// stop the video worker
	if g.videoCh != nil {
		g.v.stop()
		close(g.videoCh)
		<-g.videoDone
	}

	// we rebuild new pipeline every time because
	// changing the piplne parts proves to be unreliable
	g.v.destroy()
	if err := g.initVideo(); err != nil {
		return fmt.Errorf("gst video reinit: %w", err)
	}

	g.keyI = 0
	return nil
}

func buildVideoPipeline(w, h, scaledW, scaledH int, pixFmt uint32, scaleMethod string,
	maxThreads int, conf config.Encoder, log *logger.Logger, fps float64, vfr bool) (*pipe, error) {
	format := cmp.Or(pixFmtToGst[pixFmt], "RGBA")
	kfi := cmp.Or(conf.Video.KeyframeInterval, 120)
	opts, codec := conf.VideoSettings()
	if opts == nil {
		return nil, fmt.Errorf("unsupported codec: %s", conf.Video.Codec)
	}

	build := builder{}
	build.Grow(512)
	build.write("appsrc name=video_src is-live=true ")
	var fn, fd C.gint
	C.doubleToFraction(C.gdouble(fps), &fn, &fd)
	if vfr {
		build.writef(`caps="video/x-raw,format=%s,width=%d,height=%d,framerate=0/1,max-framerate=%d/%d" `, format, w, h, int(fn), int(fd))
	} else {
		build.writef(`caps="video/x-raw,format=%s,width=%d,height=%d,framerate=%d/%d" `, format, w, h, int(fn), int(fd))
	}
	build.writef("! videoconvertscale name=video_scale chroma-resampler=cubic method=%s n-threads=2 ", cmp.Or(scaleMethod, "nearest-neighbour"))
	build.write("! queue name=video_q max-size-buffers=1 ")
	build.writef("! capsfilter name=video_caps caps=\"video/x-raw,format=I420,color-range=0_255,width=%d,height=%d ", scaledW, scaledH)
	if conf.Video.Colorimetry != "" {
		build.writef(",colorimetry=%s", conf.Video.Colorimetry)
	}
	build.write("\" ")
	build.write("! ")
	switch codec {
	// h264 - requires gstreamer1.0-plugins-ugly + x264 (GPL)
	// case "h264":
	// 	return fmt.Sprintf("%s threads=%d key-int-max=%d",
	// 		cmp.Or(gst.Encoder, "x264enc"),
	// 		cmp.Or(conf.Video.Threads, 0),
	// 		kfi,
	// 	)
	case "vp8", "vp9":
		encoder := cmp.Or(opts.Encoder, fmt.Sprintf("vp%cenc", codec[len(codec)-1]))
		threads := cmp.Or(conf.Video.Threads, cpuCores)
		if maxThreads > 0 && threads > maxThreads {
			threads = maxThreads
		}
		build.writef("%s name=video_enc threads=%d keyframe-max-dist=%d ", encoder, threads, kfi)
	}
	build.writeIfSet(opts.Params)
	if opts.Caps != "" {
		build.write(" ! " + opts.Caps)
	}
	build.write(" ! queue name=video_q2 max-size-buffers=1 ")
	build.write("! appsink name=video_sink sync=false max-buffers=1 drop=true")

	pipeDef := build.String()

	log.Debug().Msgf("Gstreamer [video]: %s", pipeDef)

	pipeline, err := gst.NewPipelineFromString(pipeDef)
	if err != nil {
		return nil, err
	}
	srcEl, err := pipeline.GetElementByName("video_src")
	if err != nil {
		return nil, err
	}
	sinkEl, err := pipeline.GetElementByName("video_sink")
	if err != nil {
		return nil, err
	}
	go watchBus(pipeline, log)
	return &pipe{
		pipeline: pipeline,
		source:   app.SrcFromElement(srcEl),
		sink:     app.SinkFromElement(sinkEl),
		srcPad:   srcEl.GetStaticPad("src"),
	}, nil
}

func buildAudioPipeline(srcHz int, conf config.Encoder, log *logger.Logger) (*pipe, error) {
	opts := conf.AudioSettings()
	if opts == nil {
		return nil, fmt.Errorf("unsupported audio codec: %s", conf.Audio.Codec)
	}

	build := builder{}
	build.Grow(384)
	build.write("appsrc name=audio_src is-live=true ")
	build.writef("caps=audio/x-raw,format=S16LE,rate=%d,channels=2,layout=interleaved ", srcHz)
	if srcHz != 48_000 {
		m, q := "kaiser", 10
		switch conf.Audio.Resampler {
		case 0:
			m, q = "nearest", 0
		case 1:
			m, q = "linear", 4
		}
		build.write("! audioresample sinc-filter-mode=full ")
		build.writef("resample-method=%s quality=%d ! audio/x-raw,rate=48000 ", m, q)
	}
	build.writef("! %s name=audio_enc %s ", cmp.Or(opts.Encoder, "opusenc"), opts.Params)
	build.write("! queue name=audio_q max-size-buffers=1 ")
	build.write("! appsink name=audio_sink sync=false max-buffers=1 drop=true")

	pipeDef := build.String()
	log.Debug().Msgf("Gstreamer [audio]: %s", pipeDef)

	pipeline, err := gst.NewPipelineFromString(pipeDef)
	if err != nil {
		return nil, err
	}
	srcEl, _ := pipeline.GetElementByName("audio_src")
	sinkEl, _ := pipeline.GetElementByName("audio_sink")
	encEl, _ := pipeline.GetElementByName("audio_enc")

	frameMs := defaultAudioFrameMs
	if encEl != nil {
		if v, e := encEl.GetProperty("frame-size"); e == nil {
			if ms := v.(int); ms > 0 {
				frameMs = time.Duration(ms) * time.Millisecond
			}
		}
	}

	go watchBus(pipeline, log)
	return &pipe{
		pipeline: pipeline,
		source:   app.SrcFromElement(srcEl),
		sink:     app.SinkFromElement(sinkEl),
		srcPad:   srcEl.GetStaticPad("src"),
		frameDur: frameMs,
	}, nil
}

func gstVideoFormat(format string) uint32 {
	if v, ok := pixFmtCache[format]; ok {
		return v
	}
	pix := unsafe.Pointer(unsafe.StringData(format + "\x00"))
	var p runtime.Pinner
	p.Pin(pix)
	defer p.Unpin()
	v := uint32(C.gstVideoFormatFromString((*C.gchar)(pix)))
	pixFmtCache[format] = v
	return v
}

func round(x int, scale float64) int { return (int(float64(x)*scale) + 1) &^ 1 }

func watchBus(pipeline *gst.Pipeline, log *logger.Logger) {
	bus := pipeline.GetPipelineBus()

	for {
		msg := bus.TimedPop(gst.ClockTimeNone)
		if msg == nil {
			return
		}
		switch msg.Type() {
		case gst.MessageError:
			gerr := msg.ParseError()
			log.Error().Str("debug", gerr.DebugString()).Err(gerr).Msg("gst pipeline error")
		case gst.MessageWarning:
			gerr := msg.ParseWarning()
			if strings.Contains(gerr.Error(), "invalid video buffer") {
				log.Debug().Str("debug", gerr.DebugString()).Err(gerr).Msg("gst pipeline warning")
			} else {
				log.Warn().Str("debug", gerr.DebugString()).Err(gerr).Msg("gst pipeline warning")
			}
		case gst.MessageEOS:
			return
		default:
			// log.Debug().Msg("gst pipeline message: " + msg.String())
		}
	}
}
