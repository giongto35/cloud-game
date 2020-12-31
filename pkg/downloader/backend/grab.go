package backend

import (
	"log"

	"github.com/cavaliercoder/grab"
)

type GrabDownloader struct {
	client      *grab.Client
	concurrency int
}

func NewGrabDownloader() GrabDownloader {
	return GrabDownloader{
		client:      grab.NewClient(),
		concurrency: 5,
	}
}

func (d GrabDownloader) Request(dest string, urls ...string) (files []string) {
	reqs := make([]*grab.Request, 0)
	for _, url := range urls {
		req, err := grab.NewRequest(dest, url)
		if err != nil {
			log.Printf("error: couldn't make request URL: %v, %v", url, err)
		} else {
			reqs = append(reqs, req)
		}
	}

	// check each response
	for resp := range d.client.DoBatch(d.concurrency, reqs...) {
		if err := resp.Err(); err != nil {
			log.Printf("error: download failed: %v\n", err)
		} else {
			log.Printf("Downloaded [%v] %s\n", resp.HTTPResponse.Status, resp.Filename)
			files = append(files, resp.Filename)
		}
	}
	return
}
