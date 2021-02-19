package nanoarch

import (
	"math/rand"
	"testing"
)

func TestConcurrentInput(t *testing.T) {
	players := NewPlayerSessionInput()

	session := "mad-test-session"
	events := 1000
	go func() {
		for i := 0; i < events*2; i++ {
			player := rand.Intn(controllersNum)
			go players.session.setInput(session, player, 100, []byte{})
			// here it usually crashes
			go players.session.close(session)
		}
	}()
	go func() {
		for i := 0; i < events*2; i++ {
			player := rand.Intn(controllersNum)
			go players.isKeyPressed(uint(player), 100)
		}
	}()
}
