package remotehttp

import (
	"log"
	"os"

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

	arch    core.ArchInfo
	repo    repo.Repository
	altRepo repo.Repository
	client  downloader.Downloader
	fmu     *flock.Flock
}

func NewRemoteHttpManager(conf emulator.LibretroConfig) Manager {
	repoConf := conf.Cores.Repo.Main
	altRepoConf := conf.Cores.Repo.Secondary
	// used for synchronization of multiple process
	fileLock := conf.Cores.Repo.ExtLock
	if fileLock == "" {
		fileLock = os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"
	}

	arch, err := core.GetCoreExt()
	if err != nil {
		log.Printf("error: %v", err)
	}

	m := Manager{
		BasicManager: manager.BasicManager{Conf: conf},
		arch:         arch,
		client:       downloader.NewDefaultDownloader(),
		fmu:          flock.New(fileLock),
	}

	if repoConf.Type != "" {
		m.repo = repo.New(repoConf.Type, repoConf.Url, repoConf.Compression, "buildbot")
	}
	if altRepoConf.Type != "" {
		m.altRepo = repo.New(altRepoConf.Type, altRepoConf.Url, altRepoConf.Compression, "")
	}

	return m
}

func (m *Manager) Sync() error {
	// IPC lock if multiple worker processes on the same machine
	m.fmu.Lock()
	defer m.fmu.Unlock()

	download := diff(m.Conf.GetCores(), m.GetInstalled())
	if failed := m.download(download); len(failed) > 0 {
		log.Printf("[core-dl] error: unable to download these cores: %v", failed)
	}
	return nil
}

func (m *Manager) getCoreUrls(names []string, repo repo.Repository) (urls []backend.Download) {
	for _, c := range names {
		urls = append(urls, backend.Download{Key: c, Address: repo.GetCoreUrl(c, m.arch)})
	}
	return
}

func (m *Manager) download(cores []emulator.CoreInfo) (failed []string) {
	if len(cores) == 0 || m.repo == nil {
		return
	}
	var prime, second []string
	for _, n := range cores {
		if !n.AltRepo {
			prime = append(prime, n.Name)
		} else {
			second = append(second, n.Name)
		}
	}
	log.Printf("[core-dl] <<< download | main: %v | alt: %v", prime, second)
	unavailable := m.down(prime, m.repo)
	if len(unavailable) > 0 && m.altRepo != nil {
		log.Printf("[core-dl] error: unable to download some cores, trying 2nd repository")
		failed = append(failed, m.down(unavailable, m.altRepo)...)
	}
	if m.altRepo != nil {
		unavailable := m.down(second, m.altRepo)
		if len(unavailable) > 0 {
			log.Printf("[core-dl] error: unable to download some cores, trying 1st repository")
			failed = append(failed, m.down(unavailable, m.repo)...)
		}
	}
	return
}

func (m *Manager) down(cores []string, repo repo.Repository) (failed []string) {
	if len(cores) == 0 || repo == nil {
		return
	}
	_, failed = m.client.Download(m.Conf.GetCoresStorePath(), m.getCoreUrls(cores, repo)...)
	return
}

// diff returns a list of not installed cores.
func diff(declared, installed []emulator.CoreInfo) (diff []emulator.CoreInfo) {
	if len(declared) == 0 {
		return
	}
	if len(installed) == 0 {
		return declared
	}
	v := map[string]struct{}{}
	for _, x := range installed {
		v[x.Name] = struct{}{}
	}
	for _, x := range declared {
		if _, ok := v[x.Name]; !ok {
			diff = append(diff, x)
		}
	}
	return
}
