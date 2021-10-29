package backend

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/cavaliercoder/grab"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type GrabDownloader struct {
	client      *grab.Client
	parallelism int
	log         *logger.Logger
}

func NewGrabDownloader() GrabDownloader {
	return GrabDownloader{
		client: &grab.Client{
			UserAgent: "Cloud-Game/2.2",
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					Proxy:           http.ProxyFromEnvironment,
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			},
		},
		parallelism: 5,
	}
}

func (d GrabDownloader) Request(dest string, urls ...Download) (ok []string, nook []string) {
	reqs := make([]*grab.Request, 0)
	for _, url := range urls {
		req, err := grab.NewRequest(dest, url.Address)
		if err != nil {
			log.Printf("error: couldn't make request URL: %v, %v", url, err)
		} else {
			req.Label = url.Key
			reqs = append(reqs, req)
		}
	}

	// check each response
	for resp := range d.client.DoBatch(d.parallelism, reqs...) {
		r := resp.Request
		if err := resp.Err(); err != nil {
			log.Printf("error: download [%s] %s failed: %v\n", r.Label, r.URL(), err)
			if resp.HTTPResponse == nil || resp.HTTPResponse.StatusCode == 404 {
				nook = append(nook, resp.Request.Label)
			}
		} else {
			log.Printf("Downloaded [%v] [%s] %v -> %s", resp.HTTPResponse.Status, r.Label, r.URL(), resp.Filename)
			ok = append(ok, resp.Filename)
		}
	}
	return
}
