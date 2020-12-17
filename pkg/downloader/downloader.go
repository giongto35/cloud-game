package downloader

import "log"

type Down struct {
	backend Downloader
	pipe    []Process
}

type Downloader interface {
	Download(dest string, urls ...string) []string
}

type Process func(string, []string) []string

func (d *Down) Down(dest string, urls ...string) []string {
	log.Printf("+++++++++++++++++++++++++++++++++++++==")
	files := d.backend.Download(dest, urls...)

	for _, op := range d.pipe {
		files = op(dest, files)
	}

	return files
}
