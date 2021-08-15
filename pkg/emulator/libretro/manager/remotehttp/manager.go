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
	"github.com/gofrs/flock"
)

type Manager struct {
	manager.BasicManager

	arch   core.ArchInfo
	repo   repo.Repository
	client downloader.Downloader
	fmu    *flock.Flock
}

func NewRemoteHttpManager(conf emulator.LibretroConfig) Manager {
	repoConf := conf.Cores.Repo.Main
	// used for synchronization of multiple process
	fileLock := conf.Cores.Repo.ExtLock
	if fileLock == "" {
		fileLock = os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"
	}

	arch, err := core.GetCoreExt()
	if err != nil {
		log.Printf("error: %v", err)
	}

	return Manager{
		BasicManager: manager.BasicManager{
			Conf: conf,
		},
		arch:   arch,
		repo:   repo.New(repoConf.Type, repoConf.Url, repoConf.Compression, "buildbot"),
		client: downloader.NewDefaultDownloader(),
		fmu:    flock.New(fileLock),
	}
}

func (m *Manager) Sync() error {
	declared := m.Conf.GetCores()

	// IPC lock if multiple worker processes on the same machine
	m.fmu.Lock()
	defer m.fmu.Unlock()

	installed := m.GetInstalled()
	download := diff(declared, installed)

	_, failed := m.download(download)
	if len(failed) > 0 {
		log.Printf("[core-dl] error: unable to download some cores, trying 2nd repository")
		conf := m.Conf.Cores.Repo.Secondary
		if conf.Type != "" {
			if fallback := repo.New(conf.Type, conf.Url, conf.Compression, ""); fallback != nil {
				defer m.setRepo(m.repo)
				m.setRepo(fallback)
				_, _ = m.download(failed)
			}
		}
	}

	return nil
}

func (m *Manager) getCoreUrls(names []string, repo repo.Repository) (urls []backend.Download) {
	for _, c := range names {
		urls = append(urls, backend.Download{Key: c, Address: repo.GetCoreUrl(c, m.arch)})
	}
	return
}

func (m *Manager) setRepo(repo repo.Repository) {
	m.repo = repo
}

func (m *Manager) download(cores []string) (succeeded []string, failed []string) {
	if len(cores) > 0 && m.repo != nil {
		dir := m.Conf.GetCoresStorePath()
		log.Printf("[core-dl] <<< download: %v", strings.Join(cores, ", "))
		_, failed = m.client.Download(dir, m.getCoreUrls(cores, m.repo)...)
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
