package games

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Config is an external configuration
type Config struct {
	// some directory which is going to be
	// the root folder for the library
	BasePath string
	// a list of supported file extensions
	Supported []string
	// a list of ignored words in the files
	Ignored []string
	// print some additional info
	Verbose bool
	// enable directory changes watch
	WatchMode bool
}

// libConf is an optimized internal library configuration
type libConf struct {
	path      string
	supported map[string]bool
	ignored   map[string]bool
	verbose   bool
	watchMode bool
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

	// to restrict parallel execution
	// or throttling
	// !CAS would be better
	mu                sync.Mutex
	isScanning        bool
	isScanningDelayed bool
}

type GameLibrary interface {
	GetAll() []GameMetadata
	FindGameByName(name string) GameMetadata
	Scan()
}

type FileExtensionWhitelist interface {
	GetSupportedExtensions() []string
}

type GameMetadata struct {
	uid string
	// the display name of the game
	Name string
	// the game file extension (e.g. nes, n64)
	Type string
	Base string
	// the game path relative to the library base path
	Path string
}

func (c Config) GetSupportedExtensions() []string { return c.Supported }

func NewLib(conf Config) GameLibrary { return NewLibWhitelisted(conf, conf) }

func NewLibWhitelisted(conf Config, filter FileExtensionWhitelist) GameLibrary {
	hasSource := true
	dir, err := filepath.Abs(conf.BasePath)
	if err != nil {
		hasSource = false
		log.Printf("[lib] invalid source: %v (%v)\n", conf.BasePath, err)
	}

	if len(conf.Supported) == 0 {
		conf.Supported = filter.GetSupportedExtensions()
	}

	library := &library{
		config: libConf{
			path:      dir,
			supported: toMap(conf.Supported),
			ignored:   toMap(conf.Ignored),
			verbose:   conf.Verbose,
			watchMode: conf.WatchMode,
		},
		mu:        sync.Mutex{},
		games:     map[string]GameMetadata{},
		hasSource: hasSource,
	}

	if conf.WatchMode && hasSource {
		go library.watch()
	}

	return library
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

func (lib *library) Scan() {
	if !lib.hasSource {
		log.Printf("[lib] scan... skipped (no source)\n")
		return
	}

	// scan throttling
	lib.mu.Lock()
	if lib.isScanning {
		defer lib.mu.Unlock()
		lib.isScanningDelayed = true
		log.Printf("[lib] scan... delayed\n")
		return
	}
	lib.isScanning = true
	lib.mu.Unlock()

	log.Printf("[lib] scan... started\n")

	start := time.Now()
	var games []GameMetadata
	dir := lib.config.path
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info != nil && !info.IsDir() && lib.isFileExtensionSupported(path) {
			meta := getMetadata(path, dir)
			meta.uid = hash(path)

			if !lib.config.ignored[meta.Name] {
				games = append(games, meta)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("[lib] scan error with %q: %v\n", dir, err)
	}

	if len(games) > 0 {
		lib.set(games)
	}

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

	log.Printf("[lib] scan... completed\n")
}

// watch adds the ability to rescan the entire library
// during filesystem changes in a watched directory.
// !to add incremental library change
func (lib *library) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[lib] watcher has failed: %v", err)
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
		log.Printf("[lib] watch error %v", err)
	}
	<-done
	_ = watcher.Close()
	log.Printf("[lib] the watch has ended\n")
}

func (lib *library) set(games []GameMetadata) {
	res := make(map[string]GameMetadata)
	for _, value := range games {
		res[value.Name] = value
	}
	lib.games = res
}

func (lib *library) isFileExtensionSupported(path string) bool {
	ext := filepath.Ext(path)
	if ext == "" {
		return false
	}
	return lib.config.supported[ext[1:]]
}

// getMetadata returns game info from a path
func getMetadata(path string, basePath string) GameMetadata {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	relPath, _ := filepath.Rel(basePath, path)

	return GameMetadata{
		Name: strings.TrimSuffix(name, ext),
		Type: ext[1:],
		Path: relPath,
	}
}

// dumpLibrary printouts the current library snapshot of games
func (lib *library) dumpLibrary() {
	var gameList strings.Builder
	for _, game := range lib.games {
		gameList.WriteString("    " + game.Name + " (" + game.Path + ")" + "\n")
	}

	log.Printf("\n"+
		"--------------------------------------------\n"+
		"--- The Library of ROMs                  ---\n"+
		"--------------------------------------------\n"+
		"%v"+
		"--------------------------------------------\n"+
		"--- ROMs: %03d %26s ---\n"+
		"--------------------------------------------\n",
		gameList.String(), len(lib.games), lib.lastScanDuration)
}

// hash makes an MD5 hash of the string
func hash(str string) string {
	h := md5.New()
	_, err := io.WriteString(h, str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func toMap(list []string) map[string]bool {
	res := make(map[string]bool)
	for _, s := range list {
		res[s] = true
	}
	return res
}
