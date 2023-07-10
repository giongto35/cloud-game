package github

import (
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/arch"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/buildbot"
)

type RepoGithub struct {
	buildbot.RepoBuildbot
}

func NewGithubRepo(address string, compression string) RepoGithub {
	return RepoGithub{RepoBuildbot: buildbot.NewBuildbotRepo(address, compression)}
}

func (r RepoGithub) GetCoreUrl(file string, info arch.Info) string {
	return r.RepoBuildbot.GetCoreUrl(file, info) + "?raw=true"
}
