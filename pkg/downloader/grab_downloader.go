package downloader

import (
	"log"
	"os"
	"time"

	"github.com/cavaliercoder/grab"
)

type GrabDownloader struct {
	conf Config
}

func NewGrabDownloader(conf Config) Downloader {
	return &GrabDownloader{
		conf: conf,
	}
}

func (d *GrabDownloader) Download(url string, dest string) {
	client := grab.NewClient()
	req, _ := grab.NewRequest(dest, url)

	log.Printf("Downloading %v...\n", req.URL())
	resp := client.Do(req)
	log.Printf("  %v\n", resp.HTTPResponse.Status)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			log.Printf("  transferred %v / %v bytes (%.2f%%)\n",
				resp.BytesComplete(),
				resp.Size(),
				100*resp.Progress())

		case <-resp.Done:
			// download is complete
			break Loop
		}
	}

	// check for errors
	if err := resp.Err(); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("Download failed: %v\n", err)
		log.SetOutput(os.Stdout)
		os.Exit(1)
	}

	log.Printf("Download saved to ./%v \n", resp.Filename)
}
