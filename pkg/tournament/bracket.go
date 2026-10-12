// Package tournament runs single elimination tournaments: random brackets,
// a FIFO execution queue limited by a semaphore (concurrent matches) and the
// automatic advance of winners.
package tournament

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/store"
)

// Match statuses.
const (
	StatusPending  = "pending"  // waiting for players from the previous round
	StatusReady    = "ready"    // both players known, waiting in the execution queue
	StatusStarting = "starting" // a server is reserved, players were called
	StatusRunning  = "running"  // the game room is live
	StatusFinished = "finished"
	StatusStalled  = "stalled" // nobody showed up: the admin decides
)

// Tournament statuses.
const (
	TRegistering = "registering"
	TRunning     = "running"
	TFinished    = "finished"
	TCanceled    = "canceled"
)

// Result sources.
const (
	SourceRAM      = "ram"
	SourceAdmin    = "admin"
	SourceWalkover = "walkover"
)

// size returns the bracket size: the next power of two (at least 2).
func size(n int) int {
	s := 2
	for s < n {
		s <<= 1
	}
	return s
}

// Rounds returns the number of rounds for n players.
func Rounds(n int) int {
	r := 0
	for s := size(n); s > 1; s >>= 1 {
		r++
	}
	return r
}

// RoundName returns the pt-BR name of a round (1-based) out of total.
func RoundName(round, total int) string {
	switch total - round {
	case 0:
		return "Final"
	case 1:
		return "Semifinal"
	case 2:
		return "Quartas de final"
	case 3:
		return "Oitavas de final"
	}
	return "Rodada de " + strconv.Itoa(1<<(total-round+1))
}

// Shuffle returns a random permutation of the players (crypto/rand).
func Shuffle(players []int64) []int64 {
	out := append([]int64(nil), players...)
	for i := len(out) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			panic(err)
		}
		out[i], out[int(j.Int64())] = out[int(j.Int64())], out[i]
	}
	return out
}

// Build creates every match of a single elimination bracket for the (already
// shuffled) players. Byes are spread so no first round match is empty: a
// player without an opponent wins by walkover. Later rounds start pending.
func Build(players []int64) []*store.Match {
	n := len(players)
	if n < 2 {
		return nil
	}
	sz := size(n)
	rounds := Rounds(n)
	half := sz / 2
	now := time.Now().UTC().Truncate(time.Second)
	var out []*store.Match
	for p := 0; p < half; p++ {
		a := players[p]
		m := &store.Match{Round: 1, Position: p, P1: &a, Status: StatusReady, ReadyAt: &now}
		if half+p < n {
			b := players[half+p]
			m.P2 = &b
		} else {
			m.Status, m.Source, m.WinnerId, m.FinishedAt, m.ReadyAt = StatusFinished, SourceWalkover, &a, &now, nil
		}
		out = append(out, m)
	}
	for r := 2; r <= rounds; r++ {
		for p := 0; p < sz>>r; p++ {
			out = append(out, &store.Match{Round: r, Position: p, Status: StatusPending})
		}
	}
	return out
}
