package backend

import (
	"crypto/tls"
	"net/http"

	"github.com/cavaliercoder/grab"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type GrabDownloader struct {
	client      *grab.Client
	parallelism int
	log         *logger.Logger
}

func NewGrabDownloader(log *logger.Logger) GrabDownloader {
	return GrabDownloader{
		client: &grab.Client{
			UserAgent: "Cloud-Game/2.0",
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					Proxy:           http.ProxyFromEnvironment,
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			},
		},
		parallelism: 5,
		log:         log,
	}
}

func (d GrabDownloader) Request(dest string, urls ...Download) (ok []string, nook []string) {
	reqs := make([]*grab.Request, 0)
	for _, url := range urls {
		req, err := grab.NewRequest(dest, url.Address)
		if err != nil {
			d.log.Error().Err(err).Msgf("couldn't make request URL: %v, %v", url, err)
		} else {
			req.Label = url.Key
			reqs = append(reqs, req)
		}
	}

	// check each response
	for resp := range d.client.DoBatch(d.parallelism, reqs...) {
		r := resp.Request
		if err := resp.Err(); err != nil {
			d.log.Error().Err(err).Msgf("download [%s] %s has failed: %v", r.Label, r.URL(), err)
			if resp.HTTPResponse.StatusCode == 404 {
				nook = append(nook, resp.Request.Label)
			}
		} else {
			d.log.Info().Msgf("Downloaded [%v] [%s] -> %s", resp.HTTPResponse.Status, r.Label, resp.Filename)
			ok = append(ok, resp.Filename)
		}
	}
	return
}
