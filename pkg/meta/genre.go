package meta

import (
	"regexp"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/store"
)

// Genre translates a libretro genre into the catalog category (pt-BR).
func Genre(raw string) string {
	if g, ok := genres[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return g
	}
	if raw == "" {
		return ""
	}
	return "Outros"
}

var genres = map[string]string{
	"action":              "Ação",
	"platform":            "Plataforma",
	"shooter":             "Tiro",
	"lightgun shooter":    "Tiro",
	"shoot'em up":         "Nave (shmup)",
	"fighting":            "Luta",
	"beat'em up":          "Briga de rua",
	"racing":              "Corrida",
	"sports":              "Esportes",
	"sports with animals": "Esportes",
	"hunting and fishing": "Esportes",
	"role-playing (rpg)":  "RPG",
	"adventure":           "Aventura",
	"puzzle":              "Puzzle",
	"quiz":                "Puzzle",
	"casual game":         "Puzzle",
	"board":               "Tabuleiro e cartas",
	"card":                "Tabuleiro e cartas",
	"gambling":            "Tabuleiro e cartas",
	"strategy":            "Estratégia",
	"simulation":          "Simulação",
	"educational":         "Educativo",
	"music / dancing":     "Música",
	"pinball":             "Outros",
	"compilation":         "Outros",
	"various":             "Outros",
	"adult":               "Outros",
}

// Categories is the list of catalog categories, in display order.
var Categories = []string{"Ação", "Aventura", "Plataforma", "RPG", "Luta", "Briga de rua", "Tiro", "Nave (shmup)", "Corrida", "Esportes", "Puzzle", "Estratégia", "Simulação", "Tabuleiro e cartas", "Música", "Educativo", "Outros"}

var hints = []struct {
	re    *regexp.Regexp
	genre string
}{
	{regexp.MustCompile(`(?i)\b(soccer|football|baseball|basketball|nba|nhl|tennis|golf|hockey|boxing|wrestling|olympic|skate|surf|snowboard|fifa|madden)\b`), "Esportes"},
	{regexp.MustCompile(`(?i)\b(kart|racing|race|racer|rally|grand prix|drift|motocross)\b`), "Corrida"},
	{regexp.MustCompile(`(?i)\b(kombat|fighter|fighters|fighting|tekken)\b`), "Luta"},
	{regexp.MustCompile(`(?i)\b(rpg|quest|saga|chronicles|phantasy|tactics)\b`), "RPG"},
	{regexp.MustCompile(`(?i)\b(puzzle|tetris|columns|bubble|bomberman|picross)\b`), "Puzzle"},
	{regexp.MustCompile(`(?i)\b(mario|sonic|donkey kong|kirby|mega ?man|rockman|earthworm|contra)\b`), "Plataforma"},
	{regexp.MustCompile(`(?i)\b(chess|poker|casino|cards?|solitaire|shogi|mahjong)\b`), "Tabuleiro e cartas"},
	{regexp.MustCompile(`(?i)\b(wars?|empire|civilization|aerobiz|tycoon)\b`), "Estratégia"},
}

func guessGenre(title string) string {
	for _, h := range hints {
		if h.re.MatchString(title) {
			return h.genre
		}
	}
	return ""
}

// overrides holds catalog data for games the public databases don't know
// (homebrew, demos, DOS and arcade titles). Keys are lower-case title prefixes.
var overrides = map[string]store.GameMeta{
	"rogue":            {Genre: "RPG", Year: 1980, Developer: "Michael Toy, Glenn Wichman, Ken Arnold", Players: 1},
	"anguna":           {Genre: "Aventura", Year: 2014, Developer: "Bite The Chili Productions", Players: 1},
	"sushi the cat":    {Genre: "Plataforma", Year: 2016, Developer: "Homebrew", Players: 1},
	"sample demo by":   {Genre: "Outros", Developer: "Florian (domínio público)", Players: 1},
	"alwa's awakening": {Genre: "Plataforma", Year: 2018, Developer: "Elden Pixels", Players: 1},
}

func override(title string) (store.GameMeta, bool) {
	t := strings.ToLower(title)
	for k, v := range overrides {
		if strings.HasPrefix(t, k) {
			return v, true
		}
	}
	return store.GameMeta{}, false
}
