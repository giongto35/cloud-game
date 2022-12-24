package worker

import (
	"os"

	"github.com/giongto35/cloud-game/v2/pkg/worker/storage"
)

type CloudSaveRoom struct {
	GamingRoom
	storage storage.CloudStorage // a cloud storage to store room state online
}

func WithCloudStorage(room GamingRoom, storage storage.CloudStorage) *CloudSaveRoom {
	cr := CloudSaveRoom{
		GamingRoom: room,
		storage:    storage,
	}
	if err := room.(*CloudSaveRoom).Download(); err != nil {
		room.GetLog().Warn().Err(err).Msg("The room is not in the cloud")
	}
	return &cr
}

func (c *CloudSaveRoom) Download() error {
	// saveOnlineRoomToLocal save online room to local.
	// !Supports only one file of main save state.

	data, err := c.storage.Load(c.GetId())
	if err != nil {
		return err
	}
	// Save the data fetched from a cloud provider to the local server
	if data != nil {
		if err := os.WriteFile(c.GetEmulator().GetHashPath(), data, 0644); err != nil {
			return err
		}
		c.GetLog().Debug().Msg("Successfully downloaded cloud save")
	}
	return nil
}

func (c *CloudSaveRoom) HasSave() bool {
	_, err := c.storage.Load(c.GetId())
	if err == nil {
		return true
	}
	return c.GamingRoom.HasSave()
}

func (c *CloudSaveRoom) SaveGame() error {
	if err := c.GamingRoom.SaveGame(); err != nil {
		return err
	}
	if err := c.storage.Save(c.GetId(), c.GetEmulator().GetHashPath()); err != nil {
		return err
	}
	c.GetLog().Debug().Msg("Cloud save is successful")
	return nil
}
