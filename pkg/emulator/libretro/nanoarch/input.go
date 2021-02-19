package nanoarch

import "sync"

const (
	// how many axes on the D-pad
	dpadAxesNum = 4
	// the upper limit on how many controllers (players)
	// are possible for one play session (emulator instance)
	controllersNum = 8
)

const (
	InputTerminate = 0xFFFF
)

type Players struct {
	session playerSession
}

type playerSession struct {
	sync.RWMutex

	state map[string][]controllerState
}

type controllerState struct {
	keyState uint16
	axes     [dpadAxesNum]int16
}

func NewPlayerSessionInput() Players {
	return Players{
		session: playerSession{
			state: map[string][]controllerState{},
		},
	}
}

// close terminates user input session.
func (ps *playerSession) close(id string) {
	ps.Lock()
	defer ps.Unlock()

	delete(ps.state, id)
}

// setInput sets input state for some player in a game session.
func (ps *playerSession) setInput(id string, player int, buttons uint16, dpad []byte) {
	ps.Lock()
	defer ps.Unlock()

	if _, ok := ps.state[id]; !ok {
		ps.state[id] = make([]controllerState, controllersNum)
	}

	ps.state[id][player].keyState = buttons
	for i, axes := 0, len(dpad); i < dpadAxesNum && (i+1)*2+1 < axes; i++ {
		axis := (i + 1) * 2
		ps.state[id][player].axes[i] = int16(dpad[axis+1])<<8 + int16(dpad[axis])
	}
}

// isKeyPressed checks if some button is pressed by any player.
func (p *Players) isKeyPressed(player uint, key int) (pressed bool) {
	p.session.RLock()
	defer p.session.RUnlock()

	for k := range p.session.state {
		if ((p.session.state[k][player].keyState >> uint(key)) & 1) == 1 {
			return true
		}
	}
	return
}

// isDpadTouched checks if D-pad is used by any player.
func (p *Players) isDpadTouched(player uint, axis uint) (shift int16) {
	p.session.RLock()
	defer p.session.RUnlock()

	for k := range p.session.state {
		value := p.session.state[k][player].axes[axis]
		if value != 0 {
			return value
		}
	}
	return
}

type InputEvent struct {
	RawState  []byte
	PlayerIdx int
	ConnID    string
}

func (ie InputEvent) bitmap() uint16 { return uint16(ie.RawState[1])<<8 + uint16(ie.RawState[0]) }
