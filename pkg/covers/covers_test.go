package covers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

func TestBestMatch(t *testing.T) {
	tests := []struct {
		title string
		names []string
		want  string
	}{
		{"rogue", []string{"Doom - The Roguelike (2005)", "Rogue (1983)", "Rogue", "Sea Rogue (1992)"}, "Rogue"},
		{"Alwa's Awakening (Demo)", []string{
			"Alwa's Awakening - The 8-Bit Edition (World) (Aftermarket) (Unl)",
			"Alwa's Awakening - The 8-Bit Edition (World) (Demo) (Aftermarket) (Unl)",
			"Alwa's Awakening Soundtrack (World) (Aftermarket) (Unl)",
		}, "Alwa's Awakening - The 8-Bit Edition (World) (Demo) (Aftermarket) (Unl)"},
		{"Pokemon - Emerald Version (U)", []string{
			"Pokemon - Emerald Version (Japan)",
			"Pokemon - Emerald Version (USA, Europe)",
			"Pokemon - Emerald Version (USA, Europe) [T-Pt]",
		}, "Pokemon - Emerald Version (USA, Europe)"},
		{"Mario & Luigi", []string{"Mario and Luigi (USA)"}, "Mario and Luigi (USA)"},
		{"Prince of Persia 2 - The Shadow & The Flame (USA)", []string{"Prince of Persia 2 - The Shadow _ The Flame (USA)"}, "Prince of Persia 2 - The Shadow _ The Flame (USA)"},
		{"The Ignition Factor", []string{"Ignition Factor, The (USA)", "Ignition Factor, The (Japan)"}, "Ignition Factor, The (USA)"},
	}
	for _, tt := range tests {
		m, ok := bestMatch(tt.title, tt.names)
		if !ok || m.name != tt.want {
			t.Errorf("%q: got %q (%v), want %q", tt.title, m.name, ok, tt.want)
		}
	}
	// short titles must not match by prefix
	if _, ok := bestMatch("Ys", []string{"Ys Book I & II (USA)"}); ok {
		t.Error("short titles must match exactly")
	}
	if _, ok := bestMatch("Sushi The Cat", []string{"Hagane no Renkinjutsushi (Japan)"}); ok {
		t.Error("unrelated titles must not match")
	}
}

// fake libretro server
func fakeServer(t *testing.T, hits *atomic.Int32) *httptest.Server {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("fake")...)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/DOS/Named_Boxarts/":
			_, _ = w.Write([]byte(`<a href="?C=N">x</a><a href="/DOS/">up</a><a href="Rogue%20(1983).png">r</a><a href="Rogue.png">r</a>`))
		case "/DOS/Named_Boxarts/Rogue.png":
			_, _ = w.Write(png)
		case "/DOS/Named_Titles/", "/DOS/Named_Snaps/":
			_, _ = w.Write([]byte(``))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestServiceDownloadsOnceAndStores(t *testing.T) {
	var hits atomic.Int32
	srv := fakeServer(t, &hits)
	defer srv.Close()

	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	dir := t.TempDir()
	s := New(st, config.Covers{Enabled: true, Dir: dir, Url: srv.URL}, logger.Default())
	ctx := context.Background()

	// concurrent first requests share one lookup
	var wg sync.WaitGroup
	paths := make([]string, 5)
	for i := range paths {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p, err := s.Path(ctx, "dos", "rogue")
			if err != nil {
				t.Error(err)
			}
			paths[i] = p
		}(i)
	}
	wg.Wait()
	if paths[0] == "" || !strings.HasPrefix(paths[0], dir) {
		t.Fatalf("unexpected path %q", paths[0])
	}
	if b, err := os.ReadFile(paths[0]); err != nil || len(b) < 8 {
		t.Fatalf("image not saved: %v", err)
	}
	before := hits.Load()

	// a fresh service (restart) uses the stored image without the network
	s2 := New(st, config.Covers{Enabled: true, Dir: dir, Url: srv.URL}, logger.Default())
	if p, err := s2.Path(ctx, "dos", "rogue"); err != nil || p != paths[0] {
		t.Fatalf("expected cached image, got %q %v", p, err)
	}
	if hits.Load() != before {
		t.Fatal("cached covers must not hit the network")
	}

	// unknown games are remembered as missing
	if _, err := s.Path(ctx, "dos", "Jogo Inexistente"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	before = hits.Load()
	if _, err := s.Path(ctx, "dos", "Jogo Inexistente"); !errors.Is(err, ErrNotFound) || hits.Load() != before {
		t.Fatal("missing covers must be cached too")
	}
	found, missing, _ := s.Stats(ctx)
	if found != 1 || missing != 1 {
		t.Fatalf("stats: %d found, %d missing", found, missing)
	}

	// systems without a libretro database are simply missing
	if _, err := s.Path(ctx, "unknown", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for unknown system, got %v", err)
	}
}

func TestNetworkErrorsAreNotCached(t *testing.T) {
	st, _ := store.Open(":memory:")
	defer func() { _ = st.Close() }()
	s := New(st, config.Covers{Dir: t.TempDir(), Url: "http://127.0.0.1:1"}, logger.Default())
	if _, err := s.Path(context.Background(), "dos", "rogue"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a network error, got %v", err)
	}
	if _, err := st.Cover(context.Background(), Key("dos", "rogue")); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("network errors must not be stored as missing")
	}
}
