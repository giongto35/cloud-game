package repo

import (
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/arch"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/buildbot"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/github"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/raw"
)

type (
	Data struct {
		Url         string
		Compression string
	}

	Repository interface {
		GetCoreUrl(file string, info arch.Info) (url string)
	}
)

func New(kind string, url string, compression string, defaultRepo string) Repository {
	var repository Repository
	switch kind {
	case "raw":
		repository = raw.NewRawRepo(url)
	case "github":
		repository = github.NewGithubRepo(url, compression)
	case "buildbot":
		repository = buildbot.NewBuildbotRepo(url, compression)
	default:
		if defaultRepo != "" {
			repository = New(defaultRepo, url, compression, "")
		}
	}
	return repository
}
