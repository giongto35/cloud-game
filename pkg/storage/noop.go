package storage

import "errors"

type NoopCloudStorage struct{}

var noopErr = errors.New("an empty storage stub")

func NewNoopCloudStorage() (*NoopCloudStorage, error) {
	return nil, noopErr
}

func (n *NoopCloudStorage) Save(_ string, _ string) (err error) {
	return nil
}

func (n *NoopCloudStorage) Load(_ string) (data []byte, err error) {
	return nil, noopErr
}
