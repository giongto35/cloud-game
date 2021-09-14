package storage

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"

	"cloud.google.com/go/storage"
)

type GoogleCloudClient struct {
	bucket *storage.BucketHandle
	ctx    context.Context
}

// NewGoogleCloudClient returns a Google Cloud Storage client or nil if the client is not initialized.
func NewGoogleCloudClient() (*GoogleCloudClient, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("warn: failed to create Google Cloud Storage client: %v", err)
		return nil, err
	}
	bucket := client.Bucket("game-save")
	return &GoogleCloudClient{bucket: bucket}, nil
}

// Save saves a file to GCS.
func (c *GoogleCloudClient) Save(name string, srcFile string) (err error) {
	// Bypass if client is nil
	if c == nil {
		return nil
	}

	reader, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	wc := c.bucket.Object(name).NewWriter(c.ctx)
	if _, err = io.Copy(wc, reader); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

// Load loads file from GCS.
func (c *GoogleCloudClient) Load(name string) (data []byte, err error) {
	// Bypass if client is nil
	if c == nil {
		return nil, errors.New("cloud storage was not initialized")
	}

	rc, err := c.bucket.Object(name).NewReader(c.ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rc.Close()
	}()

	data, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}
