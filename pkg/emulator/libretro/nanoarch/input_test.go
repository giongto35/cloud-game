package nanoarch

import (
	"math/rand"
	"sync"
	"testing"
)

func TestConcurrentInput(t *testing.T) {
	players := NewGameSessionInput()

	events := 1000
	var wg sync.WaitGroup

	wg.Add(events * 2)

	go func() {
		for i := 0; i < events; i++ {
			player := rand.Intn(maxPort)
			go func() {
				players.setInput(player, []byte{0})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < events; i++ {
			player := rand.Intn(maxPort)
			go func() {
				players.isKeyPressed(uint(player), 100)
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}
