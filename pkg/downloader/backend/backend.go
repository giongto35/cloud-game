package backend

type Download struct {
	Key     string
	Address string
}

type Client interface {
	Request(dest string, urls ...Download) ([]string, []string)
}
