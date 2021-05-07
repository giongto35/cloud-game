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

// TODO: Add interface, abstract out Gstorage
type Client struct {
	bucket  *storage.BucketHandle
	gclient *storage.Client
}

// NewInitClient returns nil of client is not initialized
func NewInitClient() *Client {
	bucketName := "game-save"

	client, err := NewClient(bucketName)
	if err != nil {
		log.Printf("Warn: Failed to create client: %v", err)
	} else {
		log.Println("Online storage is initialized")
	}

	return client
}

// NewClient inits a new Client accessing to GCP
func NewClient(bucketName string) (*Client, error) {
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.

	// Creates a client.
	gclient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	// Creates a Bucket instance.
	bucket := gclient.Bucket(bucketName)

	return &Client{
		bucket:  bucket,
		gclient: gclient,
	}, nil
}

// Savefile save srcFile to GCP
func (c *Client) SaveFile(name string, srcFile string) (err error) {
	// Bypass if client is nil
	if c == nil {
		return nil
	}

	reader, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	// Copy source file to GCP
	wc := c.bucket.Object(name).NewWriter(context.Background())
	if _, err = io.Copy(wc, reader); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

// Loadfile loads file from GCP
func (c *Client) LoadFile(name string) (data []byte, err error) {
	// Bypass if client is nil
	if c == nil {
		return nil, errors.New("cloud storage was not initialized")
	}

	rc, err := c.bucket.Object(name).NewReader(context.Background())
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}
