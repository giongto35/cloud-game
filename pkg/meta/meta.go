// Package meta fills the game catalog (genre, year, developer, players) from the free
// libretro-database (https://github.com/libretro/libretro-database), so the library can
// be searched and filtered. Every lookup is stored in the database and made only once.
package meta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/covers"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

var ErrNotFound = errors.New("game metadata not found")

const (
	datTTL         = 30 * 24 * time.Hour
	maxDatBytes    = 32 << 20
	requestTimeout = 60 * time.Second
)

// kinds of metadat databases that carry one field each (the PlayStation database has all in one file).
var kinds = []string{"genre", "releaseyear", "developer", "maxusers"}

// systemDats maps emulator systems (config.yaml keys) to libretro database names.
var systemDats = map[string][]string{
	"gba":  {"Nintendo - Game Boy Advance", "Nintendo - Game Boy Color", "Nintendo - Game Boy"},
	"gbc":  {"Nintendo - Game Boy Color", "Nintendo - Game Boy"},
	"nes":  {"Nintendo - Nintendo Entertainment System"},
	"snes": {"Nintendo - Super Nintendo Entertainment System"},
	"n64":  {"Nintendo - Nintendo 64"},
	"pcsx": {"Sony - PlayStation"},
}

// richKinds are databases that already hold every field in a single file.
var richKinds = map[string][]string{"pcsx": {"developer"}}

type record struct {
	genre, developer, publisher, description string
	year, players                            int
}

type Service struct {
	store *store.Store
	dir   string
	base  string
	retry time.Duration
	http  *http.Client
	log   *logger.Logger

	mu       sync.Mutex
	index    map[string]*index // by system
	inflight map[string]*sync.WaitGroup
}

// filter lists the titles whose record satisfies has.
func (x *index) filter(has func(*record) bool) []string {
	var out []string
	for _, n := range x.names {
		if has(x.recs[n]) {
			out = append(out, n)
		}
	}
	return out
}

type index struct {
	names []string
	recs  map[string]*record
	at    time.Time
}

func New(st *store.Store, conf config.Meta, log *logger.Logger) *Service {
	retry := time.Duration(conf.RetryDays) * 24 * time.Hour
	if retry <= 0 {
		retry = 7 * 24 * time.Hour
	}
	dir := conf.Dir
	if dir == "" {
		dir = "./data/meta"
	}
	base := strings.TrimRight(conf.Url, "/")
	if base == "" {
		base = "https://raw.githubusercontent.com/libretro/libretro-database/master/metadat"
	}
	return &Service{
		store: st, dir: dir, base: base, retry: retry, log: log,
		http:     &http.Client{Timeout: requestTimeout},
		index:    map[string]*index{},
		inflight: map[string]*sync.WaitGroup{},
	}
}

func Key(system, title string) string { return strings.ToLower(system) + "|" + title }

// Info returns the catalog data of a game, looking it up the first time.
func (s *Service) Info(ctx context.Context, system, title string) (*store.GameMeta, error) {
	key := Key(system, title)
	for {
		if m, ok := s.cached(ctx, key); ok {
			if m == nil {
				return nil, ErrNotFound
			}
			return m, nil
		}
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
			return nil, err
		}
	}
}

// cached answers from the database; ok=false means a lookup is needed.
func (s *Service) cached(ctx context.Context, key string) (*store.GameMeta, bool) {
	m, err := s.store.GameMeta(ctx, key)
	if err != nil {
		return nil, false
	}
	switch m.Status {
	case store.MetaFound:
		return m, true
	case store.MetaMissing:
		if time.Since(m.UpdatedAt) < s.retry {
			return nil, true
		}
	}
	return nil, false
}

func (s *Service) resolve(ctx context.Context, key, system, title string) error {
	m := store.GameMeta{Key: key, System: system, Title: title, Status: store.MetaFound}

	idx, err := s.load(ctx, strings.ToLower(system))
	if err != nil {
		return err // network trouble: don't remember it as "missing"
	}
	if idx != nil {
		var best record
		var found bool
		// the best title may exist only in some of the databases: complete the gaps
		// from the best title that has the missing field
		for _, has := range []func(*record) bool{
			func(*record) bool { return true },
			func(r *record) bool { return r.genre != "" },
			func(r *record) bool { return r.year != 0 },
			func(r *record) bool { return r.developer != "" },
			func(r *record) bool { return r.players != 0 },
		} {
			names := idx.names
			if !found {
				// first pass: any title
			} else {
				names = idx.filter(has)
			}
			if i, ok := covers.MatchTitle(title, names); ok {
				r := idx.recs[names[i]]
				found = true
				if best.genre == "" {
					best.genre = r.genre
				}
				if best.year == 0 {
					best.year = r.year
				}
				if best.developer == "" {
					best.developer = r.developer
				}
				if best.publisher == "" {
					best.publisher = r.publisher
				}
				if best.players == 0 {
					best.players = r.players
				}
				if best.description == "" {
					best.description = r.description
				}
			}
			if best.genre != "" && best.year != 0 && best.developer != "" && best.players != 0 {
				break
			}
		}
		if found {
			m.GenreRaw, m.Genre = best.genre, Genre(best.genre)
			m.Year, m.Developer, m.Publisher = best.year, best.developer, best.publisher
			m.Players, m.Description = best.players, best.description
			m.Source = "libretro-database"
		}
	}
	// gaps (and systems without a database) are completed by hand-made data and title hints
	if o, ok := override(title); ok {
		fill(&m, o)
		m.Source = "belko"
	}
	if m.Genre == "" {
		if g := guessGenre(title); g != "" {
			m.Genre, m.Source = g, firstNonEmpty(m.Source, "heuristic")
		}
	}
	if m.Genre == "" && m.Year == 0 && m.Developer == "" && m.Players == 0 {
		m.Status = store.MetaMissing
		s.log.Info().Str("game", title).Str("system", system).Msg("meta: not available")
	} else {
		s.log.Info().Str("game", title).Str("genre", m.Genre).Int("year", m.Year).Msg("meta: saved")
	}
	return s.store.SetGameMeta(ctx, m)
}

func fill(m *store.GameMeta, o store.GameMeta) {
	if o.Genre != "" {
		m.Genre = o.Genre
	}
	if o.Year != 0 && m.Year == 0 {
		m.Year = o.Year
	}
	if o.Developer != "" && m.Developer == "" {
		m.Developer = o.Developer
	}
	if o.Players != 0 && m.Players == 0 {
		m.Players = o.Players
	}
	if o.Description != "" && m.Description == "" {
		m.Description = o.Description
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// load builds (once per TTL) the title index of a system. nil index: the system has no database.
func (s *Service) load(ctx context.Context, system string) (*index, error) {
	dats := systemDats[system]
	if len(dats) == 0 {
		return nil, nil
	}
	s.mu.Lock()
	idx, ok := s.index[system]
	s.mu.Unlock()
	if ok && time.Since(idx.at) < datTTL {
		return idx, nil
	}

	k := kinds
	if rk, ok := richKinds[system]; ok {
		k = rk
	}
	recs := map[string]*record{}
	for _, name := range dats {
		for _, kind := range k {
			body, err := s.dat(ctx, kind, name)
			if err != nil {
				return nil, err
			}
			if len(body) > 0 {
				merge(recs, parseDat(string(body)))
			}
		}
	}
	idx = &index{recs: recs, at: time.Now()}
	for n := range recs {
		idx.names = append(idx.names, n)
	}
	sort.Strings(idx.names)
	s.mu.Lock()
	s.index[system] = idx
	s.mu.Unlock()
	s.log.Info().Str("system", system).Int("entries", len(idx.names)).Msg("meta: database loaded")
	return idx, nil
}

// dat returns a metadat file (disk cache, then download); empty body: the file doesn't exist.
func (s *Service) dat(ctx context.Context, kind, name string) ([]byte, error) {
	cache := filepath.Join(s.dir, "_dats", kind+"_"+strings.NewReplacer(" ", "_", "/", "_").Replace(name)+".dat")
	if st, err := os.Stat(cache); err == nil && time.Since(st.ModTime()) < datTTL {
		if b, err := os.ReadFile(cache); err == nil {
			return b, nil
		}
	}
	u := s.base + "/" + kind + "/" + url.PathEscape(name) + ".dat"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BelkoArcade/1.0 (+catalog)")
	res, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == http.StatusNotFound {
		_ = writeFile(cache, nil)
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meta: %s: %s", u, res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxDatBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxDatBytes {
		return nil, fmt.Errorf("meta: response too big: %s", u)
	}
	_ = writeFile(cache, data)
	return data, nil
}

type entry map[string]string

// parseDat reads the clrmamepro text format: game ( key "value" ... rom ( ... ) ).
func parseDat(text string) []entry {
	var out []entry
	var cur entry
	depth := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case depth == 0 && strings.HasPrefix(line, "game ("):
			cur, depth = entry{}, 1
		case depth == 0:
			if strings.HasSuffix(line, "(") {
				depth = -1 // header block: skip until it closes
			}
		case depth == -1:
			if line == ")" {
				depth = 0
			}
		case line == ")":
			depth--
			if depth == 0 {
				out = append(out, cur)
				cur = nil
			}
		case strings.HasSuffix(line, "("):
			depth++
		case depth == 1:
			// values are quoted ("Action") or bare (users 2)
			if i := strings.IndexByte(line, ' '); i > 0 {
				v := strings.TrimSpace(line[i:])
				switch {
				case len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"':
					cur[line[:i]] = v[1 : len(v)-1]
				case v != "" && !strings.ContainsAny(v, "()"):
					cur[line[:i]] = v
				}
			}
		}
	}
	return out
}

func merge(recs map[string]*record, entries []entry) {
	for _, e := range entries {
		name := firstNonEmpty(e["comment"], e["name"])
		if name == "" {
			continue
		}
		r := recs[name]
		if r == nil {
			r = &record{}
			recs[name] = r
		}
		if v := e["genre"]; v != "" {
			r.genre = v
		}
		if v := e["developer"]; v != "" {
			r.developer = v
		}
		if v := e["publisher"]; v != "" {
			r.publisher = v
		}
		if v := e["description"]; v != "" {
			r.description = v
		}
		if v := e["releaseyear"]; v != "" {
			r.year, _ = strconv.Atoi(v)
		}
		if v := firstNonEmpty(e["users"], e["maxusers"]); v != "" {
			r.players, _ = strconv.Atoi(strings.TrimRight(v, "+"))
		}
	}
}

// Warm looks up in the background the data of a game library.
func (s *Service) Warm(games map[string]string) {
	go func() {
		for title, system := range games {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			if _, err := s.Info(ctx, system, title); err != nil && !errors.Is(err, ErrNotFound) {
				s.log.Warn().Err(err).Str("game", title).Msg("meta: prefetch failed")
			}
			cancel()
		}
	}()
}

// All lists the catalog data stored so far.
func (s *Service) All(ctx context.Context) ([]store.GameMeta, error) { return s.store.AllGameMeta(ctx) }

// Retry forgets the stored results (and the downloaded databases) so they are looked up again.
func (s *Service) Retry(ctx context.Context) error {
	s.mu.Lock()
	s.index = map[string]*index{}
	s.mu.Unlock()
	_ = os.RemoveAll(filepath.Join(s.dir, "_dats"))
	return s.store.ForgetGameMeta(ctx, true)
}

func (s *Service) Stats(ctx context.Context) (found, missing int, err error) {
	return s.store.GameMetaStats(ctx)
}

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
