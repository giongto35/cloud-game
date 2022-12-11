package remotehttp

import (
	"os"

	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/manager"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/repo"
	"github.com/gofrs/flock"
)

type Manager struct {
	manager.BasicManager

	arch    core.ArchInfo
	repo    repo.Repository
	altRepo repo.Repository
	client  Downloader
	fmu     *flock.Flock
	log     *logger.Logger
}

func NewRemoteHttpManager(conf emulator.LibretroConfig, log *logger.Logger) Manager {
	repoConf := conf.Cores.Repo.Main
	altRepoConf := conf.Cores.Repo.Secondary
	// used for synchronization of multiple process
	fileLock := conf.Cores.Repo.ExtLock
	if fileLock == "" {
		fileLock = os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"
	}
	log.Debug().Msgf("Using .lock file: %v", fileLock)

	arch, err := core.GetCoreExt()
	if err != nil {
		log.Error().Err(err).Msg("couldn't get Libretro core file extension")
	}

	m := Manager{
		BasicManager: manager.BasicManager{Conf: conf},
		arch:         arch,
		client:       NewDefaultDownloader(log),
		fmu:          flock.New(fileLock),
		log:          log,
	}

	if repoConf.Type != "" {
		m.repo = repo.New(repoConf.Type, repoConf.Url, repoConf.Compression, "buildbot")
	}
	if altRepoConf.Type != "" {
		m.altRepo = repo.New(altRepoConf.Type, altRepoConf.Url, altRepoConf.Compression, "")
	}

	return m
}

func CheckCores(conf emulator.Emulator, log *logger.Logger) error {
	if !conf.Libretro.Cores.Repo.Sync {
		return nil
	}
	log.Info().Msg("Starting Libretro cores sync...")
	coreManager := NewRemoteHttpManager(conf.Libretro, log)
	// make a dir for cores
	dir := coreManager.Conf.GetCoresStorePath()
	if err := os.MkdirAll(dir, os.ModeDir); err != nil {
		return err
	}
	if err := coreManager.Sync(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Sync() error {
	// IPC lock if multiple worker processes on the same machine
	m.fmu.Lock()
	defer m.fmu.Unlock()

	installed, err := m.GetInstalled()
	if err != nil {
		return err
	}
	download := diff(m.Conf.GetCores(), installed)
	if failed := m.download(download); len(failed) > 0 {
		m.log.Warn().Msgf("[core-dl] error: unable to download these cores: %v", failed)
	}
	return nil
}

func (m *Manager) getCoreUrls(names []string, repo repo.Repository) (urls []Download) {
	for _, c := range names {
		urls = append(urls, Download{Key: c, Address: repo.GetCoreUrl(c, m.arch)})
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
	m.log.Info().Msgf("[core-dl] <<< download | main: %v | alt: %v", prime, second)
	primeFails := m.down(prime, m.repo)
	if len(primeFails) > 0 && m.altRepo != nil {
		m.log.Warn().Msgf("[core-dl] error: unable to download some cores, trying 2nd repository")
		failed = append(failed, m.down(primeFails, m.altRepo)...)
	}
	if m.altRepo != nil {
		altFails := m.down(second, m.altRepo)
		if len(altFails) > 0 {
			m.log.Error().Msgf("[core-dl] error: unable to download some cores, trying 1st repository")
			failed = append(failed, m.down(altFails, m.repo)...)
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
