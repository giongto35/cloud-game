package storage

import "errors"

type NoopCloudStorage struct{}

var noopErr = errors.New("an empty storage stub")

func NewNoopCloudStorage() (*NoopCloudStorage, error) {
	return nil, noopErr
}

func (n *NoopCloudStorage) Save(name string, localPath string) (err error) {
	return nil
}

func (n *NoopCloudStorage) Load(name string) (data []byte, err error) {
	return nil, noopErr
}
