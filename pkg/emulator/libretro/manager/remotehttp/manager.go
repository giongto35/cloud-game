package remotehttp

import (
	"log"
	"os"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/downloader"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/buildbot"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/github"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/repo/raw"
	"github.com/gofrs/flock"
)

type Manager struct {
	manager.BasicManager

	repo   repo.Repository
	client downloader.Downloader
	fmu    *flock.Flock
}

func NewRemoteHttpManager(conf emulator.LibretroConfig) Manager {
	repoConf := conf.Cores.Repo

	var repository repo.Repository
	switch repoConf.Compression {
	case "raw":
		repository = raw.NewRawRepo(repoConf.Url)
	case "github":
		repository = github.NewGithubRepo()
	case "buildbot":
		buildbot.NewBuildbotRepo(repoConf.Url, repoConf.Compression)
		fallthrough
	default:
	}

	// used for synchronization of multiple process
	fileLock := os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"

	return Manager{
		BasicManager: manager.BasicManager{
			Conf: conf,
		},
		repo:   repository,
		client: downloader.NewDefaultDownloader(),
		fmu:    flock.New(fileLock),
	}
}

func (m Manager) Sync() error {
	declared := m.Conf.GetCores()
	dir := m.Conf.GetCoresStorePath()

	// IPC lock if multiple worker processes on the same machine
	m.fmu.Lock()
	defer m.fmu.Unlock()

	installed := m.GetInstalled()
	download := diff(installed, declared)

	if len(download) > 0 {
		log.Printf("Start download Libretro cores: %v", strings.Join(download, ", "))
		m.client.Download(dir, m.getCoreUrls(download)...)
	}
	return nil
}

func (m Manager) getCoreUrls(names []string) (urls []string) {
	arch, _ := core.GetCoreExt()
	for _, c := range names {
		urls = append(urls, m.repo.GetCoreData(c, arch).Url)
	}
	return
}

// diff returns a list of not installed cores.
func diff(declared, installed []string) (diff []string) {
	v := map[string]struct{}{}
	for _, x := range declared {
		v[x] = struct{}{}
	}
	for _, x := range installed {
		if _, ok := v[x]; !ok {
			diff = append(diff, x)
		}
	}
	return
}
