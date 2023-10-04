package recorder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const demuxFile = "input.txt"

// createFfmpegMuxFile makes FFMPEG concat demuxer file.
//
// ffmpeg concat demuxer, see: https://ffmpeg.org/ffmpeg-formats.html#concat
// example:
//
// !to change
//
//	ffmpeg -f concat -i input.txt \
//		   -ac 2 -channel_layout stereo -i audio.wav \
//		   -b:a 192K -crf 23 -vf fps=30 -pix_fmt yuv420p \
//		   out.mp4
func createFfmpegMuxFile(dir string, fPattern string, frameTimes []time.Duration, opts Options) (er error) {
	demux, err := newFile(dir, demuxFile)
	if err != nil {
		return err
	}
	defer func() { er = demux.Close() }()

	b := strings.Builder{}

	b.WriteString("ffconcat version 1.0\n")
	b.WriteString(meta("v", "1"))
	b.WriteString(meta("date", time.Now().Format("20060102")))
	b.WriteString(meta("game", opts.Game))
	b.WriteString(meta("fps", opts.Fps))
	b.WriteString(meta("freq", opts.Frequency))
	b.WriteString(meta("pix", opts.Pix))
	_, err = demux.WriteString(fmt.Sprintf("%s\n", b.String()))
	if err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	i := 0
	sync := opts.Vsync && len(frameTimes) > 0
	ext := filepath.Ext(fPattern)
	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			continue
		}
		dur := 1 / opts.Fps
		if sync && i < len(frameTimes) {
			dur = frameTimes[i].Seconds()
			if dur == 0 {
				dur = 1 / opts.Fps
			}
			i++
		}
		w, h, s := ExtractFileInfo(file.Name())
		inf := fmt.Sprintf("file %v\nduration %f\n%s%s%s", name, dur,
			metaf("width", w), metaf("height", h), metaf("stride", s))
		if _, err := demux.WriteString(inf); err != nil {
			er = err
		}
	}
	if err = demux.Flush(); err != nil {
		er = err
	}
	return er
}

// meta adds stream_meta key value line.
func meta(key string, value any) string { return fmt.Sprintf("stream_meta %s '%v'\n", key, value) }

// metaf adds file_packet_meta key value line.
func metaf(key string, value any) string {
	return fmt.Sprintf("file_packet_meta %s '%v'\n", key, value)
}
