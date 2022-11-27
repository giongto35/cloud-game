package nanoarch

import "sync"

const (
	maxPort     = 4
	dpadAxesNum = 4
	KeyPressed  = 1
	KeyReleased = 0
)

type (
	GameSessionInput struct {
		state [maxPort]controller
		mu    sync.RWMutex
	}
	controller struct {
		keys uint16
		axes [dpadAxesNum]int16
	}
)

func NewGameSessionInput() GameSessionInput { return GameSessionInput{state: [maxPort]controller{}} }

// close terminates user input session.
func (p *GameSessionInput) close() {
	//p.mu.Lock()
	//
	//delete(p., id)
	//p.mu.Unlock()
}

// setInput sets input state for some player in a game session.
func (p *GameSessionInput) setInput(player int, buttons uint16, dpad []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.state[player].keys = buttons
	for i, axes := 0, len(dpad); i < dpadAxesNum && (i+1)*2+1 < axes; i++ {
		axis := (i + 1) * 2
		p.state[player].axes[i] = int16(dpad[axis+1])<<8 + int16(dpad[axis])
	}
}

// isKeyPressed checks if some button is pressed by any player.
func (p *GameSessionInput) isKeyPressed(port uint, key int) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return int((p.state[port].keys >> uint(key)) & 1)
}

// isDpadTouched checks if D-pad is used by any player.
func (p *GameSessionInput) isDpadTouched(port uint, axis uint) (shift int16) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.state[port].axes[axis]
}
