package cloud

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

type S3Client struct {
	c      *minio.Client
	bucket string
	log    *logger.Logger
}

func NewS3Client(endpoint, bucket, key, secret string, log *logger.Logger) (*S3Client, error) {
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(key, secret, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}

	exists, err := s3Client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("bucket doesn't exist")
	}

	return &S3Client{bucket: bucket, c: s3Client, log: log}, nil
}

func (s *S3Client) SetBucket(bucket string) { s.bucket = bucket }

func (s *S3Client) Save(name string, data []byte, meta map[string]string) error {
	if s == nil || s.c == nil {
		return errors.New("s3 client was not initialised")
	}
	r := bytes.NewReader(data)
	opts := minio.PutObjectOptions{
		ContentType:    "application/octet-stream",
		SendContentMd5: true,
	}
	if meta != nil {
		opts.UserMetadata = meta
	}

	info, err := s.c.PutObject(context.Background(), s.bucket, name, r, int64(len(data)), opts)
	if err != nil {
		return err
	}
	s.log.Debug().Msgf("Uploaded: %v", info)
	return nil
}

func (s *S3Client) Load(name string) (data []byte, err error) {
	if s == nil || s.c == nil {
		return nil, errors.New("s3 client was not initialised")
	}

	r, err := s.c.GetObject(context.Background(), s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, r.Close()) }()

	stats, err := r.Stat()
	log.Debug().Msgf("Downloaded: %v", stats)
	dat, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return dat, nil
}

func (s *S3Client) Has(name string) bool {
	if s == nil || s.c == nil {
		return false
	}
	_, err := s.c.StatObject(context.Background(), s.bucket, name, minio.GetObjectOptions{})
	return err == nil
}
