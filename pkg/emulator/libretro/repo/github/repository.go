package github

import (
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/buildbot"
)

type Repo struct {
	buildbot.Repo
}

func NewGithubRepo(address string, compression string) Repo {
	return Repo{Repo: buildbot.NewBuildbotRepo(address, compression)}
}

func (r Repo) GetCoreData(file string, info core.ArchInfo) repo.Data {
	dat := r.Repo.GetCoreData(file, info)
	return repo.Data{Url: dat.Url + "?raw=true", Compression: dat.Compression}
}
