package downloader

type Config struct {
}

type Downloader interface {
	Download(dest string, urls ...string)
}
