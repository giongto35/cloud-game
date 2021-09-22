package launcher

type Launcher interface {
	FindAppByName(name string) (AppMeta, error)
	ExtractAppNameFromUrl(name string) string
	GetAppNames() []string
}

type AppMeta struct {
	Name string
	Type string
	Base string
	Path string
}
