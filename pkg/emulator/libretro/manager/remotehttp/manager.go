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
	switch repoConf.Type {
	case "raw":
		repository = raw.NewRawRepo(repoConf.Url)
	case "github":
		repository = github.NewGithubRepo(repoConf.Url, repoConf.Compression)
	case "buildbot":
		fallthrough
	default:
		repository = buildbot.NewBuildbotRepo(repoConf.Url, repoConf.Compression)
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
	download := diff(declared, installed)

	if len(download) > 0 {
		log.Printf("Starting Libretro cores download: %v", strings.Join(download, ", "))
		m.client.Download(dir, m.getCoreUrls(download)...)
	}
	return nil
}

func (m Manager) getCoreUrls(names []string) (urls []string) {
	arch, err := core.GetCoreExt()
	if err != nil {
		return
	}
	for _, c := range names {
		urls = append(urls, m.repo.GetCoreData(c, arch).Url)
	}
	return
}

// diff returns a list of not installed cores.
func diff(declared, installed []string) (diff []string) {
	if len(declared) == 0 {
		return
	}

	if len(installed) == 0 {
		return declared
	}

	v := map[string]struct{}{}
	for _, x := range installed {
		v[x] = struct{}{}
	}
	for _, x := range declared {
		if _, ok := v[x]; !ok {
			diff = append(diff, x)
		}
	}
	return
}
