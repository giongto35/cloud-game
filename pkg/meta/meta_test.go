package meta

import "testing"

const sampleDat = `clrmamepro (
	name "Nintendo - Super Nintendo Entertainment System"
)

game (
	comment "3 Ninjas Kick Back (USA)"
	genre "Platform"
	rom ( crc F2EE11F9 )
)

game (
	comment "3 Ninjas Kick Back (USA)"
	users 3
)

game (
	name "Crash Team Racing (USA)"
	description "Race."
	developer "Naughty Dog"
	releaseyear "1999"
	users "4"
	genre "Racing"
	rom (
		serial "SCUS-94426"
		name "Crash Team Racing (USA).cue"
	)
)
`

func TestParseDat(t *testing.T) {
	list := parseDat(sampleDat)
	if len(list) != 3 {
		t.Fatalf("want 3 games, got %d", len(list))
	}
	recs := map[string]*record{}
	merge(recs, list)

	r := recs["3 Ninjas Kick Back (USA)"]
	if r == nil || r.genre != "Platform" || r.players != 3 {
		t.Fatalf("bad record: %+v", r)
	}
	c := recs["Crash Team Racing (USA)"]
	if c == nil || c.developer != "Naughty Dog" || c.year != 1999 || c.players != 4 || c.genre != "Racing" {
		t.Fatalf("bad record: %+v", c)
	}
}

func TestGenre(t *testing.T) {
	cases := map[string]string{
		"Platform":           "Plataforma",
		"Role-playing (RPG)": "RPG",
		"Beat'em Up":         "Briga de rua",
		"Whatever":           "Outros",
		"":                   "",
	}
	for in, want := range cases {
		if got := Genre(in); got != want {
			t.Errorf("Genre(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGuessAndOverride(t *testing.T) {
	if g := guessGenre("Super Mario Kart (USA)"); g != "Corrida" {
		t.Errorf("kart: %q", g)
	}
	if g := guessGenre("Unknown Thing"); g != "" {
		t.Errorf("unknown: %q", g)
	}
	if o, ok := override("rogue"); !ok || o.Genre != "RPG" {
		t.Errorf("override: %+v %v", o, ok)
	}
}
