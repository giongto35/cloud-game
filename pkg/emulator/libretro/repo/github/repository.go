package github

import (
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo"
)

type Repo struct {
}

func (r *Repo) GetCoreData(file string, info core.ArchInfo) repo.Data {
	return repo.Data{}
}
