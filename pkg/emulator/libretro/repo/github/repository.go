package github

import (
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/buildbot"
)

type RepoGithub struct {
	buildbot.RepoBuildbot
}

func NewGithubRepo(address string, compression string) RepoGithub {
	return RepoGithub{RepoBuildbot: buildbot.NewBuildbotRepo(address, compression)}
}

func (r RepoGithub) GetCoreUrl(file string, info core.ArchInfo) string {
	return r.RepoBuildbot.GetCoreUrl(file, info) + "?raw=true"
}
