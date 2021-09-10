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

// TODO: Add interface, abstract out Google Storage

type Client struct {
	bucket *storage.BucketHandle
	// ! not used
	client *storage.Client
}

// NewClient returns a Google Cloud Storage client or nil if the client is not initialized.
func NewClient() *Client {
	bucketName := "game-save"
	client, err := storage.NewClient(context.Background())
	if err != nil {
		log.Printf("warn: failed to create Google Cloud Storage client: %v", err)
		return nil
	}
	bucket := client.Bucket(bucketName)
	return &Client{
		bucket: bucket,
		client: client,
	}
}

// Save saves a file to GCS.
func (c *Client) Save(name string, srcFile string) (err error) {
	// Bypass if client is nil
	if c == nil {
		return nil
	}

	reader, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	wc := c.bucket.Object(name).NewWriter(context.Background())
	if _, err = io.Copy(wc, reader); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

// Load loads file from GCS.
func (c *Client) Load(name string) (data []byte, err error) {
	// Bypass if client is nil
	if c == nil {
		return nil, errors.New("cloud storage was not initialized")
	}

	rc, err := c.bucket.Object(name).NewReader(context.Background())
	if err != nil {
		return nil, err
	}
	defer func(rc *storage.Reader) {
		err = rc.Close()
	}(rc)

	data, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}
