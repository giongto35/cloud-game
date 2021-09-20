package storage

type CloudStorage interface {
	Save(name string, localPath string) (err error)
	Load(name string) (data []byte, err error)
}
