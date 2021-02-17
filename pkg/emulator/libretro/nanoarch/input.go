package nanoarch

const numAxes = 4
const maxPort = 8

const (
	InputTerminate = 0xFFFF
)

type controllerState struct {
	keyState uint16
	axes     [numAxes]int16
}

type players struct {
	session playerSession
}

type playerSession struct {
	state map[string][]controllerState
}

func NewPlayerSessionInput() players {
	return players{
		session: playerSession{
			state: map[string][]controllerState{},
		},
	}
}

func (ps *playerSession) close(id string) {
	delete(ps.state, id)
}

func (ps *playerSession) poke(id string) {
	if _, ok := ps.state[id]; !ok {
		ps.state[id] = make([]controllerState, maxPort)
	}
}

func (ps *playerSession) setInputForPlayer(id string, player int, buttons uint16, dpad []byte) {
	ps.poke(id)

	ps.state[id][player].keyState = buttons
	for i := 0; i < numAxes && (i+1)*2+1 < len(dpad); i++ {
		ps.state[id][player].axes[i] = int16(dpad[(i+1)*2+1])<<8 + int16(dpad[(i+1)*2])
	}
}

func (ps *playerSession) isKeyPressed(player uint, key int) (pressed bool) {
	for k := range ps.state {
		if ((ps.state[k][player].keyState >> uint(key)) & 1) == 1 {
			return true
		}
	}
	return
}

func (ps *playerSession) isDpad(player uint, axis uint) (shift int16) {
	for k := range ps.state {
		value := ps.state[k][player].axes[axis]
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
