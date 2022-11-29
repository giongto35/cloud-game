package nanoarch

import (
	"math/rand"
	"testing"
)

func TestConcurrentInput(t *testing.T) {
	players := NewGameSessionInput()

	events := 1000
	go func() {
		for i := 0; i < events*2; i++ {
			player := rand.Intn(maxPort)
			go players.setInput(player, []byte{0, 0})
		}
	}()
	go func() {
		for i := 0; i < events*2; i++ {
			player := rand.Intn(maxPort)
			go players.isKeyPressed(uint(player), 100)
		}
	}()
}
