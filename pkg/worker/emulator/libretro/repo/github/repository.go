package github

import (
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro/repo/buildbot"
)

type RepoGithub struct {
	buildbot.RepoBuildbot
}

func NewGithubRepo(address string, compression string) RepoGithub {
	return RepoGithub{RepoBuildbot: buildbot.NewBuildbotRepo(address, compression)}
}

func (r RepoGithub) GetCoreUrl(file string, info libretro.ArchInfo) string {
	return r.RepoBuildbot.GetCoreUrl(file, info) + "?raw=true"
}
