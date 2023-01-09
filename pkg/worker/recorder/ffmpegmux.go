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
	_, err = demux.WriteString(
		fmt.Sprintf("ffconcat version 1.0\n# v: 1\n# date: %v\n# game: %v\n# fps: %v\n# freq (hz): %v\n\n",
			time.Now().Format("20060102"), opts.Game, opts.Fps, opts.Frequency))
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
		inf := fmt.Sprintf("file %v\nduration %f\n", name, dur)
		if _, err := demux.WriteString(inf); err != nil {
			er = err
		}
	}
	if err = demux.Flush(); err != nil {
		er = err
	}
	return er
}
