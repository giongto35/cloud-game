package storage

type CloudStorage interface {
	Save(name string, localPath string) (err error)
	Load(name string) (data []byte, err error)
}

func GetCloudStorage(provider, key string) (CloudStorage, error) {
	var st CloudStorage
	var err error
	switch provider {
	case "oracle":
		st, err = NewOracleDataStorageClient(key)
	case "coordinator":
	default:
	}
	return st, err
}
