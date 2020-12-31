package downloader

import (
	"github.com/giongto35/cloud-game/v2/pkg/downloader/backend"
	"github.com/giongto35/cloud-game/v2/pkg/downloader/pipe"
)

type Downloader struct {
	backend client
	// pipe contains a sequential list of
	// operations applied to some files and
	// each operation will return a list of
	// successfully processed files
	pipe []Process
}

type client interface {
	Request(dest string, urls ...string) []string
}

type Process func(string, []string) []string

func NewDefaultDownloader() Downloader {
	return Downloader{
		backend: backend.NewGrabDownloader(),
		pipe: []Process{
			pipe.Unpack,
			pipe.Delete,
		}}
}

// Download tries to download specified with URLs list of files and
// put them into the destination folder.
// It will return a partial or full list of downloaded files,
// a list of processed files if some pipe processing functions are set.
func (d *Downloader) Download(dest string, urls ...string) []string {
	files := d.backend.Request(dest, urls...)
	for _, op := range d.pipe {
		files = op(dest, files)
	}
	return files
}
