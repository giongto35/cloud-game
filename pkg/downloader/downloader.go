package downloader

type Config struct {
}

type Downloader interface {
	Download(url string, dest string)
}
