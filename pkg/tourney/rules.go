// Package tourney holds the tournament game rules (winner detection by reading
// the emulated RAM) and the deterministic input replay format.
package tourney

import "strings"

// Rule describes how to read the result of a game from the emulated work RAM
// (offsets are relative to the start of WRAM, i.e. SNES $7E0000).
//
// All addresses were verified empirically with the RAM lab
// (pkg/worker/caged/libretro/ramlab_test.go, docs/plano/06-torneios.md).
type Rule struct {
	Key    string
	Name   string
	System string
	// Match lists lower case fragments of the ROM name that select this rule.
	Match []string
	// Score holds the per player score address (P1, P2).
	Score [2]uint32
	// Target is the score that wins the match.
	Target byte
	// TargetAddr, when non-zero, reads the target from RAM (value + TargetAdd),
	// for games where players choose it in the game menu.
	TargetAddr uint32
	TargetAdd  byte
	// Stable is how many frames the winning score must hold before it counts.
	Stable int
	// Hint is shown to the players before the match (pt-BR).
	Hint string
}

// Rules is the registry of supported tournament games.
var Rules = []Rule{
	{
		Key: "umk3", Name: "Ultimate Mortal Kombat 3", System: "snes",
		Match: []string{"ultimate mortal kombat 3", "umk3"},
		// rounds won; +1 three frames after each KO
		Score: [2]uint32{0x3AA6, 0x3AA8}, Target: 2, Stable: 30,
		Hint: "Melhor de 3 rounds. O Jogador 2 aperta Start no menu de modos para entrar na luta.",
	},
	{
		Key: "sb2", Name: "Super Bomberman 2", System: "snes",
		Match: []string{"super bomberman 2"},
		// trophies; the goal ("Troféus") is stored minus one at $7E1CA7
		Score: [2]uint32{0x5E66, 0x5E68}, TargetAddr: 0x1CA7, TargetAdd: 1, Stable: 30,
		Hint: "Modo Batalha individual com 2 jogadores. A bomba é o botão A. Vence quem chegar aos troféus combinados.",
	},
	{
		Key: "smk", Name: "Super Mario Kart", System: "snes",
		Match: []string{"super mario kart"},
		// lap byte: 128 = lap 1 ... 132 = final lap, 133 = finished
		Score: [2]uint32{0x10C1, 0x11C1}, Target: 133, Stable: 2,
		Hint: "2P Game → Match Race. Vence quem cruzar a linha de chegada primeiro.",
	},
}

// Find returns the rule for a game name, or nil.
func Find(game string) *Rule {
	g := strings.ToLower(game)
	for i := range Rules {
		for _, m := range Rules[i].Match {
			if strings.Contains(g, m) {
				return &Rules[i]
			}
		}
	}
	return nil
}

// ByKey returns the rule with the given key, or nil.
func ByKey(key string) *Rule {
	for i := range Rules {
		if Rules[i].Key == key {
			return &Rules[i]
		}
	}
	return nil
}
