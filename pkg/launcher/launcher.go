package launcher

type Launcher interface {
	FindAppByName(name string) (AppMeta, error)
	ExtractAppNameFromUrl(name string) string
}

type AppMeta struct {
	Name string
	Type string
	Path string
}
