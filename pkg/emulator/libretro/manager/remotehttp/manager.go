package remotehttp

import (
	"log"
	"os"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/downloader"
	"github.com/giongto35/cloud-game/v2/pkg/downloader/backend"
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

	arch          core.ArchInfo
	mainRepo      repo.Repository
	secondaryRepo repo.Repository
	client        downloader.Downloader
	fmu           *flock.Flock
}

func NewRemoteHttpManager(conf emulator.LibretroConfig) Manager {
	repoConfMain, repoConfSecond := conf.Cores.Repo.Main, conf.Cores.Repo.Secondary
	mainRepo := Factory(repoConfMain.Type, repoConfMain.Url, repoConfMain.Compression, "buildbot")
	var secondaryRepo repo.Repository
	if repoConfSecond.Type != "" {
		secondaryRepo = Factory(repoConfSecond.Type, repoConfSecond.Url, repoConfSecond.Compression, "")
	}

	// used for synchronization of multiple process
	fileLock := os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"

	arch, err := core.GetCoreExt()
	if err != nil {
		log.Printf("error: %v", err)
	}

	return Manager{
		BasicManager: manager.BasicManager{
			Conf: conf,
		},
		arch:          arch,
		mainRepo:      mainRepo,
		secondaryRepo: secondaryRepo,
		client:        downloader.NewDefaultDownloader(),
		fmu:           flock.New(fileLock),
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

	var failed []string
	if len(download) > 0 {
		log.Printf("Starting Libretro cores download: %v", strings.Join(download, ", "))
		_, failed = m.client.Download(dir, m.getCoreUrls(download, m.mainRepo)...)
	}
	if len(failed) > 0 && m.secondaryRepo != nil {
		log.Printf("Starting fallback Libretro cores download: %v", strings.Join(failed, ", "))
		_, failed = m.client.Download(dir, m.getCoreUrls(download, m.secondaryRepo)...)
	}

	return nil
}

func (m Manager) getCoreUrls(names []string, repo repo.Repository) (urls []backend.Download) {
	for _, c := range names {
		urls = append(urls, backend.Download{
			Key:     c,
			Address: repo.GetCoreData(c, m.arch).Url,
		})
	}
	return
}

func Factory(kind string, url string, compression string, defaultRepo string) repo.Repository {
	var repository repo.Repository
	switch kind {
	case "raw":
		repository = raw.NewRawRepo(url)
	case "github":
		repository = github.NewGithubRepo(url, compression)
	case "buildbot":
		repository = buildbot.NewBuildbotRepo(url, compression)
	default:
		if defaultRepo != "" {
			repository = Factory(defaultRepo, url, compression, "")
		}
	}
	return repository
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
