package cloud

import (
	"crypto/rand"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

func TestS3(t *testing.T) {
	t.Skip()

	name := "test"
	s3, err := NewS3Client(
		"s3.tebi.io",
		"cloudretro-001",
		"",
		"",
		logger.Default(),
	)
	if err != nil {
		t.Error(err)
	}

	buf := make([]byte, 1024*4)
	// then we can call rand.Read.
	_, err = rand.Read(buf)
	if err != nil {
		t.Error(err)
	}

	err = s3.Save(name, buf, map[string]string{"id": "test"})
	if err != nil {
		t.Error(err)
	}

	exists := s3.Has(name)
	if !exists {
		t.Errorf("don't exist, but shuld")
	}

	ne := s3.Has(name + "123213")
	if ne {
		t.Errorf("exists, but shouldn't")
	}

	dat, err := s3.Load(name)
	if err != nil {
		t.Error(err)
	}

	if len(dat) == 0 {
		t.Errorf("should be something")
	}
}
