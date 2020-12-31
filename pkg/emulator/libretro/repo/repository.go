package repo

import "github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"

type (
	Data struct {
		Url         string
		Compression CompressionType
	}

	Repository interface {
		GetCoreData(file string, info core.ArchInfo) Data
	}
)
