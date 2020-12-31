package raw

import (
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo"
)

type Repo struct {
	address     string
	compression repo.CompressionType
}

// NewRawRepo defines a simple zip file containing
// all the cores that will be extracted as is.
func NewRawRepo(address string) Repo {
	return Repo{
		address:     address,
		compression: "zip",
	}
}

func (r Repo) GetCoreData(_ string, _ core.ArchInfo) repo.Data {
	return repo.Data{Url: r.address, Compression: r.compression}
}
