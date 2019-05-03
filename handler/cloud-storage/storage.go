package storage

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"

	"cloud.google.com/go/storage"
)

type Client struct {
	bucket  *storage.BucketHandle
	gclient *storage.Client
}

func NewInitClient() *Client {
	projectID := os.Getenv("GCP_PROJECT")
	bucketName := "game-save"
	return NewClient(projectID, bucketName)
}

// NewClient inits a new Client accessing to GCP
func NewClient(projectID string, bucketName string) *Client {
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.

	// Creates a client.
	gclient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Creates a Bucket instance.
	bucket := gclient.Bucket(bucketName)

	return &Client{
		bucket:  bucket,
		gclient: gclient,
	}
}

// Savefile save srcFile to GCP
func (c *Client) SaveFile(name string, srcFile string) (err error) {
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

// Loadfile load file from GCP
func (c *Client) LoadFile(name string) (data []byte, err error) {
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
