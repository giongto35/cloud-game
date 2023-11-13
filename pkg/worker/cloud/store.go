package cloud

import (
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

type Storage interface {
	Save(name string, data []byte, tags map[string]string) (err error)
	Load(name string) (data []byte, err error)
	Has(name string) bool
}

func Store(conf config.Storage, log *logger.Logger) (Storage, error) {
	var st Storage
	var err error
	switch conf.Provider {
	case "s3":
		st, err = NewS3Client(conf.S3Endpoint, conf.S3BucketName, conf.S3AccessKeyId, conf.S3SecretAccessKey, log)
	case "coordinator":
	default:
	}
	return st, err
}
