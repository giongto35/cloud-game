package manager

import (
	"os"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/compression"
)

type Download struct {
	Key     string
	Address string
}

type Client interface {
	Request(dest string, urls ...Download) ([]string, []string)
}

type Downloader struct {
	backend Client
	// pipe contains a sequential list of
	// operations applied to some files and
	// each operation will return a list of
	// successfully processed files
	pipe []Process
	log  *logger.Logger
}

type Process func(string, []string, *logger.Logger) []string

func NewDefaultDownloader(log *logger.Logger) Downloader {
	return Downloader{
		backend: NewGrabDownloader(log),
		pipe:    []Process{unpackDelete},
		log:     log,
	}
}

// Download tries to download specified with URLs list of files and
// put them into the destination folder.
// It will return a partial or full list of downloaded files,
// a list of processed files if some pipe processing functions are set.
func (d *Downloader) Download(dest string, urls ...Download) ([]string, []string) {
	files, fails := d.backend.Request(dest, urls...)
	for _, op := range d.pipe {
		files = op(dest, files, d.log)
	}
	return files, fails
}

func unpackDelete(dest string, files []string, log *logger.Logger) []string {
	var res []string
	for _, file := range files {
		if unpack := compression.NewFromExt(file, log); unpack != nil {
			if _, err := unpack.Extract(file, dest); err == nil {
				if e := os.Remove(file); e == nil {
					res = append(res, file)
				}
			}
		}
	}
	return res
}
