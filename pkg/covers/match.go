package covers

import (
	"regexp"
	"strings"
	"unicode"
)

// Thumbnail kinds in the libretro-thumbnails repository, in order of preference.
var kinds = []string{"Named_Boxarts", "Named_Titles", "Named_Snaps"}

// systemDirs maps emulator systems (config.yaml keys) to libretro database names.
var systemDirs = map[string][]string{
	"gba":  {"Nintendo - Game Boy Advance", "Nintendo - Game Boy Color", "Nintendo - Game Boy"},
	"gbc":  {"Nintendo - Game Boy Color", "Nintendo - Game Boy"},
	"nes":  {"Nintendo - Nintendo Entertainment System"},
	"snes": {"Nintendo - Super Nintendo Entertainment System"},
	"n64":  {"Nintendo - Nintendo 64"},
	"pcsx": {"Sony - PlayStation"},
	"mame": {"FBNeo - Arcade Games", "MAME"},
	"dos":  {"DOS"},
}

var (
	reGroups = regexp.MustCompile(`\s*[(\[][^)\]]*[)\]]`)
	reTags   = regexp.MustCompile(`[(\[]([^)\]]*)[)\]]`)
)

// stop words are ignored: "Ignition Factor, The" = "The Ignition Factor",
// "Mario & Luigi" = "Mario and Luigi" (libretro also writes "&" as "_").
var stopWords = map[string]bool{"the": true, "and": true}

// normalize turns a game title into a comparable key:
// no (tags), lower case, only letters and digits, no stop words.
func normalize(s string) string {
	s = reGroups.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "'", "")
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := words[:0]
	for _, w := range words {
		if !stopWords[w] {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func tagsOf(s string) []string {
	var out []string
	for _, m := range reTags.FindAllStringSubmatch(s, -1) {
		for _, t := range strings.Split(m[1], ",") {
			out = append(out, strings.ToLower(strings.TrimSpace(t)))
		}
	}
	return out
}

var preferredRegions = map[string]int{"usa": 0, "world": 1, "europe": 2, "brazil": 3, "usa, europe": 0}

// candidate scoring: lower is better
type match struct {
	name  string
	tier  int // 0 = same title, 1 = title prefix ("Alwa's Awakening" -> "Alwa's Awakening - The 8-Bit Edition")
	score int
}

// bestMatch picks the best thumbnail name for a title among the available names.
func bestMatch(title string, names []string) (match, bool) {
	q := normalize(title)
	if q == "" {
		return match{}, false
	}
	want := tagsOf(title)
	var best match
	found := false
	for _, n := range names {
		base := normalize(n)
		tier := -1
		switch {
		case base == q:
			tier = 0
		case len(q) >= 5 && strings.HasPrefix(base, q+" "):
			tier = 1
		}
		if tier < 0 {
			continue
		}
		score := len(n) // shorter (fewer extras) is better
		tags := tagsOf(n)
		region := 9
		for _, t := range tags {
			if r, ok := preferredRegions[t]; ok && r < region {
				region = r
			}
			switch t {
			case "beta", "proto", "sample", "soundtrack":
				score += 200
			}
		}
		score += region * 20
		if strings.Contains(n, "[") { // hacks, translations, fixes
			score += 300
		}
		for _, w := range want {
			for _, t := range tags {
				if w == t {
					score -= 60 // e.g. a "(Demo)" ROM prefers the demo picture
				}
			}
		}
		m := match{name: n, tier: tier, score: score}
		if !found || m.tier < best.tier || (m.tier == best.tier && m.score < best.score) {
			best, found = m, true
		}
	}
	return best, found
}

// MatchTitle picks the best entry of names for a game title (index into names).
func MatchTitle(title string, names []string) (int, bool) {
	m, ok := bestMatch(title, names)
	if !ok {
		return 0, false
	}
	for i, n := range names {
		if n == m.name {
			return i, true
		}
	}
	return 0, false
}
