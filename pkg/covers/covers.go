// Package covers finds game cover art in the free libretro-thumbnails service
// (https://thumbnails.libretro.com) and keeps every image on the server,
// so each picture is downloaded only once.
package covers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

var ErrNotFound = errors.New("cover not found")

const (
	listingTTL     = 7 * 24 * time.Hour
	maxImageBytes  = 5 << 20
	maxListBytes   = 16 << 20
	requestTimeout = 30 * time.Second
	maxDownloads   = 3
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

var reHref = regexp.MustCompile(`href="([^"?/][^"]*\.png)"`)

type Service struct {
	store *store.Store
	dir   string
	base  string
	retry time.Duration
	http  *http.Client
	log   *logger.Logger

	sem chan struct{}

	mu       sync.Mutex
	lists    map[string]listing         // libretro dir/kind -> available names
	inflight map[string]*sync.WaitGroup // keys being resolved right now
}

type listing struct {
	names []string
	at    time.Time
}

func New(st *store.Store, conf config.Covers, log *logger.Logger) *Service {
	retry := time.Duration(conf.RetryDays) * 24 * time.Hour
	if retry <= 0 {
		retry = 7 * 24 * time.Hour
	}
	dir := conf.Dir
	if dir == "" {
		dir = "./data/covers"
	}
	base := strings.TrimRight(conf.Url, "/")
	if base == "" {
		base = "https://thumbnails.libretro.com"
	}
	return &Service{
		store: st, dir: dir, base: base, retry: retry, log: log,
		http:     &http.Client{Timeout: requestTimeout},
		sem:      make(chan struct{}, maxDownloads),
		lists:    map[string]listing{},
		inflight: map[string]*sync.WaitGroup{},
	}
}

func Key(system, title string) string { return strings.ToLower(system) + "|" + title }

func fileName(key string) string {
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:10]) + ".png"
}

// Path returns the local image of a game, downloading it the first time.
// ErrNotFound means the service has no picture for the game (the result is cached).
func (s *Service) Path(ctx context.Context, system, title string) (string, error) {
	key := Key(system, title)
	for {
		if p, err, ok := s.cached(ctx, key); ok {
			return p, err
		}
		// only one lookup per game at a time
		s.mu.Lock()
		if wg, busy := s.inflight[key]; busy {
			s.mu.Unlock()
			wg.Wait()
			continue
		}
		wg := &sync.WaitGroup{}
		wg.Add(1)
		s.inflight[key] = wg
		s.mu.Unlock()

		err := s.resolve(ctx, key, system, title)

		s.mu.Lock()
		delete(s.inflight, key)
		s.mu.Unlock()
		wg.Done()
		if err != nil {
			return "", err
		}
	}
}

// cached answers from the database/disk; ok=false means a lookup is needed.
func (s *Service) cached(ctx context.Context, key string) (string, error, bool) {
	c, err := s.store.Cover(ctx, key)
	if errors.Is(err, store.ErrNotFound) {
		return "", nil, false
	}
	if err != nil {
		return "", err, true
	}
	switch c.Status {
	case store.CoverFound:
		p := filepath.Join(s.dir, c.File)
		if _, err := os.Stat(p); err == nil {
			return p, nil, true
		}
		return "", nil, false // the file was removed: download again
	case store.CoverMissing:
		if time.Since(c.UpdatedAt) < s.retry {
			return "", ErrNotFound, true
		}
	}
	return "", nil, false
}

func (s *Service) resolve(ctx context.Context, key, system, title string) error {
	dirs := systemDirs[strings.ToLower(system)]
	var best match
	var bestUrl string
	found := false
	// try every kind (box art first) in every database of the system
	for _, kind := range kinds {
		for _, dir := range dirs {
			names, err := s.listing(ctx, dir, kind)
			if err != nil {
				return err // network trouble: don't remember it as "missing"
			}
			m, ok := bestMatch(title, names)
			if ok && (!found || m.tier < best.tier) {
				best, found = m, true
				bestUrl = s.base + "/" + url.PathEscape(dir) + "/" + kind + "/" + url.PathEscape(m.name+".png")
			}
		}
		if found && best.tier == 0 {
			break
		}
	}
	if !found {
		s.log.Info().Str("game", title).Str("system", system).Msg("cover: not available")
		return s.store.SetCover(ctx, store.Cover{Key: key, System: system, Title: title, Status: store.CoverMissing})
	}

	data, err := s.download(ctx, bestUrl, maxImageBytes)
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(data, pngMagic) {
		return fmt.Errorf("cover: not a PNG image: %s", bestUrl)
	}
	name := filepath.Join(strings.ToLower(system), fileName(key))
	if err := writeFile(filepath.Join(s.dir, name), data); err != nil {
		return err
	}
	s.log.Info().Str("game", title).Str("match", best.name).Msg("cover: saved")
	return s.store.SetCover(ctx, store.Cover{Key: key, System: system, Title: title, File: name, Source: bestUrl, Status: store.CoverFound})
}

// listing returns the picture names of a libretro database folder (cached in memory and on disk).
func (s *Service) listing(ctx context.Context, dir, kind string) ([]string, error) {
	id := dir + "/" + kind
	s.mu.Lock()
	l, ok := s.lists[id]
	s.mu.Unlock()
	if ok && time.Since(l.at) < listingTTL {
		return l.names, nil
	}

	cache := filepath.Join(s.dir, "_listings", fileName(id)+".txt")
	if st, err := os.Stat(cache); err == nil && time.Since(st.ModTime()) < listingTTL {
		if b, err := os.ReadFile(cache); err == nil {
			names := splitLines(string(b))
			s.remember(id, names, st.ModTime())
			return names, nil
		}
	}

	body, err := s.download(ctx, s.base+"/"+url.PathEscape(dir)+"/"+kind+"/", maxListBytes)
	var nf *notFoundErr
	if errors.As(err, &nf) {
		s.remember(id, nil, time.Now()) // the folder doesn't exist for this system
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, m := range reHref.FindAllStringSubmatch(string(body), -1) {
		if n, err := url.PathUnescape(m[1]); err == nil {
			names = append(names, strings.TrimSuffix(n, ".png"))
		}
	}
	_ = writeFile(cache, []byte(strings.Join(names, "\n")))
	s.remember(id, names, time.Now())
	return names, nil
}

func (s *Service) remember(id string, names []string, at time.Time) {
	s.mu.Lock()
	s.lists[id] = listing{names: names, at: at}
	s.mu.Unlock()
}

type notFoundErr struct{ url string }

func (e *notFoundErr) Error() string { return "not found: " + e.url }

func (s *Service) download(ctx context.Context, u string, limit int64) ([]byte, error) {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.sem }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BelkoArcade/1.0 (+cover cache)")
	res, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == http.StatusNotFound {
		return nil, &notFoundErr{u}
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover: %s: %s", u, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("cover: response too big: %s", u)
	}
	return data, nil
}

// Warm downloads in the background the covers of a game library.
func (s *Service) Warm(games map[string]string) {
	go func() {
		for title, system := range games {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if _, err := s.Path(ctx, system, title); err != nil && !errors.Is(err, ErrNotFound) {
				s.log.Warn().Err(err).Str("game", title).Msg("cover: prefetch failed")
			}
			cancel()
		}
	}()
}

// Retry forgets the "not found" results so they are looked up again.
func (s *Service) Retry(ctx context.Context) error {
	s.mu.Lock()
	s.lists = map[string]listing{}
	s.mu.Unlock()
	_ = os.RemoveAll(filepath.Join(s.dir, "_listings"))
	return s.store.ForgetMissingCovers(ctx)
}

func (s *Service) Stats(ctx context.Context) (found, missing int, err error) {
	return s.store.CoverStats(ctx)
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// writeFile writes atomically (temp file + rename).
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
