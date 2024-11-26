package games

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

// libConf is an optimized internal library configuration
type libConf struct {
	aliasFile   string
	path        string
	supported   map[string]struct{}
	ignored     []string
	verbose     bool
	watchMode   bool
	sessionPath string
}

type library struct {
	config libConf
	// indicates repo source existence
	hasSource bool
	// scan time
	lastScanDuration time.Duration
	// library entries
	// !should be a tree-based structure
	// game name -> game meta
	// games with duplicate names are merged
	games map[string]GameMetadata
	log   *logger.Logger

	// ids of saved games to find closed sessions
	sessions []string

	emuConf WithEmulatorInfo

	// to restrict parallel execution or throttling
	// for file watch mode
	mu                sync.Mutex
	isScanning        bool
	isScanningDelayed bool
}

type GameLibrary interface {
	GetAll() []GameMetadata
	FindGameByName(name string) GameMetadata
	Sessions() []string
	Scan()
}

type WithEmulatorInfo interface {
	GetSupportedExtensions() []string
	GetEmulator(rom string, path string) string
	SessionStoragePath() string
}

type GameMetadata struct {
	Alias  string
	Base   string
	Name   string // the display name of the game
	Path   string // the game path relative to the library base path
	System string
	Type   string // the game file extension (e.g. nes, n64)
}

func (g GameMetadata) FullPath(base string) string {
	if base == "" {
		return filepath.Join(g.Base, g.Path)
	}
	return filepath.Join(base, g.Path)
}

func NewLib(conf config.Library, emu WithEmulatorInfo, log *logger.Logger) GameLibrary {
	hasSource := true
	dir, err := filepath.Abs(conf.BasePath)
	if err != nil {
		hasSource = false
		log.Error().Err(err).Str("dir", conf.BasePath).Msg("Lib has invalid source")
	}

	if len(conf.Supported) == 0 {
		conf.Supported = emu.GetSupportedExtensions()
	}

	library := &library{
		config: libConf{
			aliasFile:   conf.AliasFile,
			path:        dir,
			supported:   toMap(conf.Supported),
			ignored:     conf.Ignored,
			verbose:     conf.Verbose,
			watchMode:   conf.WatchMode,
			sessionPath: emu.SessionStoragePath(),
		},
		mu:        sync.Mutex{},
		games:     map[string]GameMetadata{},
		hasSource: hasSource,
		log:       log,
		emuConf:   emu,
	}

	if conf.WatchMode && hasSource {
		go library.watch()
	}

	return library
}

func (lib *library) Sessions() []string {
	return lib.sessions
}

func (lib *library) GetAll() []GameMetadata {
	var res []GameMetadata
	for _, value := range lib.games {
		res = append(res, value)
	}
	return res
}

// FindGameByName returns some game info with its full filepath
func (lib *library) FindGameByName(name string) GameMetadata {
	var game GameMetadata
	if val, ok := lib.games[name]; ok {
		val.Base = lib.config.path
		return val
	}
	return game
}

func (lib *library) AliasFileMaybe() map[string]string {
	if lib.config.aliasFile == "" {
		return nil
	}

	path := filepath.Join(lib.config.path, lib.config.aliasFile)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		lib.log.Error().Msgf("couldn't open alias file, %v", err)
		return nil
	}
	defer func() { _ = file.Close() }()

	aliases := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if id, alias, found := strings.Cut(scanner.Text(), "="); found {
			aliases[id] = alias
		}
	}

	if err := scanner.Err(); err != nil {
		lib.log.Error().Msgf("alias file read error, %v", err)
	}

	return aliases
}

func (lib *library) Scan() {
	if !lib.hasSource {
		lib.log.Info().Msg("Lib scan... skipped (no source)")
		return
	}

	// scan throttling
	lib.mu.Lock()
	if lib.isScanning {
		defer lib.mu.Unlock()
		lib.isScanningDelayed = true
		lib.log.Debug().Msg("Lib scan... delayed")
		return
	}
	lib.isScanning = true
	lib.mu.Unlock()

	lib.log.Debug().Msg("Lib scan... started")

	// game name aliases
	aliases := lib.AliasFileMaybe()

	if aliases != nil {
		lib.log.Debug().Msgf("Lib game alises found")
		lib.log.Debug().Msgf(">>> %v", aliases)
	}

	start := time.Now()
	var games []GameMetadata
	dir := lib.config.path
	err := filepath.WalkDir(dir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info == nil || info.IsDir() || !lib.isExtAllowed(path) {
			return nil
		}

		meta := metadata(path, dir)
		meta.System = lib.emuConf.GetEmulator(meta.Type, meta.Path)

		if aliases != nil {
			if k, ok := aliases[meta.Name]; ok {
				meta.Alias = k
			}
		}

		ignored := false
		for _, k := range lib.config.ignored {
			if meta.Name == k {
				ignored = true
				break
			}

			if len(k) > 0 && k[0] == '.' && strings.Contains(meta.Name, k) {
				ignored = true
				break
			}
		}

		if !ignored {
			games = append(games, meta)
		}

		return nil
	})

	if err != nil {
		lib.log.Error().Err(err).Str("dir", dir).Msgf("Lib scan... failed")
		return
	}

	if len(games) > 0 {
		lib.set(games)
	}

	var sessions []string
	dir = lib.config.sessionPath
	err = filepath.WalkDir(dir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info != nil && !info.IsDir() {
			sessions = append(sessions, info.Name())
		}
		return nil
	})
	lib.sessions = sessions

	lib.lastScanDuration = time.Since(start)
	if lib.config.verbose {
		lib.dumpLibrary()
	}

	// run scan again if delayed
	lib.mu.Lock()
	defer lib.mu.Unlock()
	lib.isScanning = false
	if lib.isScanningDelayed {
		lib.isScanningDelayed = false
		go lib.Scan()
	}

	lib.log.Info().Msg("Lib scan... completed")
}

// watch adds the ability to rescan the entire library
// during filesystem changes in a watched directory.
// !to add incremental library change
func (lib *library) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		lib.log.Error().Err(err).Msg("Lib watcher has failed")
		return
	}

	done := make(chan bool)
	go func(repo *library) {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Create || event.Op == fsnotify.Remove {
					// !to try to add the proper file/dir add/remove scan logic
					// which is tricky
					repo.Scan()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}(lib)

	if err = watcher.Add(lib.config.path); err != nil {
		lib.log.Error().Err(err).Msg("Lib watch error")
	}
	<-done
	_ = watcher.Close()
	lib.log.Info().Msg("Lib watch has ended")
}

func (lib *library) set(games []GameMetadata) {
	res := make(map[string]GameMetadata)
	for _, value := range games {
		res[value.Name] = value
	}
	lib.games = res
}

func (lib *library) isExtAllowed(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	_, ok := lib.config.supported[ext[1:]]
	return ok
}

// metadata returns game info from a path
func metadata(path string, basePath string) GameMetadata {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	relPath, _ := filepath.Rel(basePath, path)

	return GameMetadata{
		Name: strings.TrimSuffix(name, ext),
		Type: strings.ToLower(ext[1:]),
		Path: relPath,
	}
}

// dumpLibrary printouts the current library snapshot of games
func (lib *library) dumpLibrary() {
	var gameList strings.Builder

	// oof
	keys := make([]string, 0, len(lib.games))
	for k := range lib.games {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		game := lib.games[k]
		alias := game.Alias
		if alias != "" {
			alias = fmt.Sprintf("[%s] ", game.Alias)
		}
		gameList.WriteString(fmt.Sprintf("    %7s   %s %s(%s)\n", game.System, game.Name, alias, game.Path))
	}

	lib.log.Debug().Msgf("Lib dump\n"+
		"--------------------------------------------\n"+
		"--- The Library of ROMs                  ---\n"+
		"--------------------------------------------\n"+
		"%v"+
		"--------------------------------------------\n"+
		"--- ROMs: %03d --- Saves: %04d %10s ---\n"+
		"--------------------------------------------",
		gameList.String(), len(lib.games), len(lib.sessions), lib.lastScanDuration)
}

func toMap(list []string) map[string]struct{} {
	res := make(map[string]struct{}, len(list))
	for _, s := range list {
		res[s] = struct{}{}
	}
	return res
}
