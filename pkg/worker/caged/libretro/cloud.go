package libretro

import (
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/cloud"
)

type CloudFrontend struct {
	Emulator
	stateName      string
	stateLocalPath string
	storage        cloud.Storage // a cloud storage to store room state online
}

func WithCloud(fe Emulator, stateName string, storage cloud.Storage) (*CloudFrontend, error) {
	r := &CloudFrontend{Emulator: fe, stateLocalPath: fe.HashPath(), stateName: stateName, storage: storage}

	// saveOnlineRoomToLocal save online room to local.
	// !Supports only one file of main save state.
	data, err := r.storage.Load(stateName)
	if err != nil {
		return nil, err
	}
	// save the data fetched from the cloud to a local directory
	if data != nil {
		if err := os.WriteFile(r.stateLocalPath, data, 0644); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (c *CloudFrontend) HasSave() bool {
	_, err := c.storage.Load(c.stateName)
	if err == nil {
		return true
	}
	return c.Emulator.HasSave()
}

func (c *CloudFrontend) SaveGameState() error {
	if err := c.Emulator.SaveGameState(); err != nil {
		return err
	}
	if err := c.storage.Save(c.stateName, c.stateLocalPath); err != nil {
		return err
	}
	return nil
}
