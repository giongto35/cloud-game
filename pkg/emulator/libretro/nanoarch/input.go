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

type controllers struct {
	state map[string][]controllerState
}

type InputEvent struct {
	RawState  []byte
	PlayerIdx int
	ConnID    string
}

func (ie InputEvent) bitmap() uint16 { return uint16(ie.RawState[1])<<8 + uint16(ie.RawState[0]) }
