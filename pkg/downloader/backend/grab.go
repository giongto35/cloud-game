package backend

import (
	"log"
	"os"

	"github.com/cavaliercoder/grab"
)

type GrabDownloader struct {
	client *grab.Client
}

func NewGrabDownloader() GrabDownloader {
	return GrabDownloader{
		client: grab.NewClient(),
	}
}

func (d GrabDownloader) Request(dest string, urls ...string) []string {
	reqs := make([]*grab.Request, 0)
	for _, url := range urls {
		req, err := grab.NewRequest(dest, url)
		if err != nil {
			panic(err)
		}
		reqs = append(reqs, req)
	}

	respch := d.client.DoBatch(4, reqs...)

	// check each response
	var files []string
	for resp := range respch {
		if err := resp.Err(); err != nil {
			log.SetOutput(os.Stderr)
			log.Printf("Download failed: %v\n", err)
			log.SetOutput(os.Stdout)
			panic(err)
		}

		log.Printf("  %v\n", resp.HTTPResponse.Status)
		log.Printf("Downloaded %s to %s\n", resp.Request.URL(), resp.Filename)
		files = append(files, resp.Filename)
	}
	return files
}
