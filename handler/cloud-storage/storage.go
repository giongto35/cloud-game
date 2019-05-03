package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"cloud.google.com/go/storage"
)

type Client struct {
	bucket  string
	gclient storage.Client
}

func NewClient() *Client {
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	projectID := "YOUR_PROJECT_ID"

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Sets the name for the new bucket.
	bucketName := "my-new-bucket"

	// Creates a Bucket instance.
	bucket := client.Bucket(bucketName)

	// Creates the new bucket.
	if err := bucket.Create(ctx, projectID, nil); err != nil {
		log.Fatalf("Failed to create bucket: %v", err)
	}

	fmt.Printf("Bucket %v created.\n", bucketName)
}

func (c *Client) SaveFile(name string, data string) (err error) {
	wc := c.gclient.Bucket(c.bucket).Object(name).NewWriter(nil)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
}

func (h *Helper) LoadFile(name string) []byte {

}
