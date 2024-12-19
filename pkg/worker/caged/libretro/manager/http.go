package manager

import (
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
)

type Manager struct {
	BasicManager

	arch    ArchInfo
	repo    Repository
	altRepo Repository
	client  Downloader
	fmu     *os.Flock
	log     *logger.Logger
}

func NewRemoteHttpManager(conf config.LibretroConfig, log *logger.Logger) Manager {
	repoConf := conf.Cores.Repo.Main
	altRepoConf := conf.Cores.Repo.Secondary

	// used for synchronization of multiple process
	flock, err := os.NewFileLock(conf.Cores.Repo.ExtLock)
	if err != nil {
		log.Error().Err(err).Msgf("couldn't make file lock")
	}

	arch, err := conf.Cores.Repo.Guess()
	if err != nil {
		log.Error().Err(err).Msg("couldn't get Libretro core file extension")
	}

	m := Manager{
		BasicManager: BasicManager{Conf: conf},
		arch:         ArchInfo(arch),
		client:       NewDefaultDownloader(log),
		fmu:          flock,
		log:          log,
	}

	if repoConf.Type != "" {
		m.repo = NewRepo(repoConf.Type, repoConf.Url, repoConf.Compression, "buildbot")
	}
	if altRepoConf.Type != "" {
		m.altRepo = NewRepo(altRepoConf.Type, altRepoConf.Url, altRepoConf.Compression, "")
	}

	return m
}

func CheckCores(conf config.Emulator, log *logger.Logger) error {
	if !conf.Libretro.Cores.Repo.Sync {
		return nil
	}
	log.Info().Msg("Starting Libretro cores sync...")
	coreManager := NewRemoteHttpManager(conf.Libretro, log)
	// make a dir for cores
	if err := os.MakeDirAll(coreManager.Conf.GetCoresStorePath()); err != nil {
		return err
	}
	if err := coreManager.Sync(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Sync() error {
	// IPC lock if multiple worker processes on the same machine
	err := m.fmu.Lock()
	if err != nil {
		m.log.Error().Err(err).Msg("file lock fail")
	}
	defer func() {
		err := m.fmu.Unlock()
		if err != nil {
			m.log.Error().Err(err).Msg("file unlock fail")
		}
	}()

	installed, err := m.GetInstalled(m.arch.Ext)
	if err != nil {
		return err
	}
	download := diff(m.Conf.GetCores(), installed)
	if failed := m.download(download); len(failed) > 0 {
		m.log.Warn().Msgf("[core-dl] error: unable to download these cores: %v", failed)
	}
	return nil
}

func (m *Manager) getCoreUrls(names []string, repo Repository) (urls []Download) {
	for _, c := range names {
		urls = append(urls, Download{Key: c, Address: repo.CoreUrl(c, m.arch)})
	}
	return
}

func (m *Manager) download(cores []config.CoreInfo) (failed []string) {
	if len(cores) == 0 || m.repo == nil {
		return
	}
	var prime, second, fail []string
	for _, n := range cores {
		if n.Name == "" {
			fail = append(fail, n.Id)
			continue
		}
		if !n.AltRepo {
			prime = append(prime, n.Name)
		} else {
			second = append(second, n.Name)
		}
	}

	if len(prime) == 0 && len(second) == 0 {
		m.log.Warn().Msgf("[core-dl] couldn't find info for %v cores, check the config", fail)
		return
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

func (m *Manager) down(cores []string, repo Repository) (failed []string) {
	if len(cores) == 0 || repo == nil {
		return
	}
	_, failed = m.client.Download(m.Conf.GetCoresStorePath(), m.getCoreUrls(cores, repo)...)
	return
}

// diff returns a list of not installed cores.
func diff(declared, installed []config.CoreInfo) (diff []config.CoreInfo) {
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
