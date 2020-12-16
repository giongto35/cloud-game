package core

type Repository interface {
	GetLink(file string, info ArchInfo) string
}
