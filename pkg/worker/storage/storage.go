package storage

type CloudStorage interface {
	Save(name string, localPath string) (err error)
	Load(name string) (data []byte, err error)
	// IsNoop shows whether a storage is no-op stub
	// !to remove when properly refactored
	IsNoop() bool
}

func GetCloudStorage(provider, key string) (CloudStorage, error) {
	var st CloudStorage
	var err error
	switch provider {
	case "oracle":
		st, err = NewOracleDataStorageClient(key)
	case "coordinator":
	default:
		st, _ = NewNoopCloudStorage()
	}
	if err != nil {
		st, _ = NewNoopCloudStorage()
	}
	return st, err
}
