package libretro

import (
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/cloud"
)

type CloudFrontend struct {
	Emulator
	uid     string
	storage cloud.Storage // a cloud storage to store room state online
}

// WithCloud adds the ability to keep game states in the cloud storage like Amazon S3.
// It supports only one file of main save state.
func WithCloud(fe Emulator, uid string, storage cloud.Storage) (*CloudFrontend, error) {
	r := &CloudFrontend{Emulator: fe, uid: uid, storage: storage}

	name := fe.SaveStateName()

	if r.storage.Has(name) {
		data, err := r.storage.Load(fe.SaveStateName())
		if err != nil {
			return nil, err
		}
		// save the data fetched from the cloud to a local directory
		if data != nil {
			if err := os.WriteFile(fe.HashPath(), data, 0644); err != nil {
				return nil, err
			}
		}
	}

	return r, nil
}

// !to use emulator save/load calls instead of the storage

func (c *CloudFrontend) HasSave() bool {
	_, err := c.storage.Load(c.SaveStateName())
	if err == nil {
		return true
	}
	return c.Emulator.HasSave()
}

func (c *CloudFrontend) SaveGameState() error {
	if err := c.Emulator.SaveGameState(); err != nil {
		return err
	}
	path := c.Emulator.HashPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return c.storage.Save(c.SaveStateName(), data, map[string]string{
		"uid":  c.uid,
		"type": "cloudretro-main-save",
	})
}
