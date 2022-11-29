package nanoarch

import (
	"sync/atomic"
	"unsafe"
)

const (
	maxPort     = 4
	dpadAxes    = 4
	KeyPressed  = 1
	KeyReleased = 0
)

// GameSessionInput stores full controller state.
// It consists of:
//   - uint16 button values
//   - int16 analog stick values
type GameSessionInput [maxPort]struct {
	keys uint32
	axes [dpadAxes]int32
}

func NewGameSessionInput() GameSessionInput {
	return [maxPort]struct {
		keys uint32
		axes [dpadAxes]int32
	}{}
}

// setInput sets input state for some player in a game session.
func (s *GameSessionInput) setInput(player int, data []byte) {
	atomic.StoreUint32(&s[player].keys, *(*uint32)(unsafe.Pointer(&data[0])))
	// tf is that
	// !to add tests
	// axis = (i+1)*2
	for i, axes := 0, len(data); i < dpadAxes && i<<1+3 < axes; i++ {
		atomic.StoreInt32(&s[player].axes[i], *(*int32)(unsafe.Pointer(&data[i<<1+2])))
		//int32(data[axis+1])<<8+int32(data[axis]))
	}
}

// isKeyPressed checks if some button is pressed by any player.
func (s *GameSessionInput) isKeyPressed(port uint, key int) int {
	return int((atomic.LoadUint32(&s[port].keys) >> uint(key)) & 1)
}

// isDpadTouched checks if D-pad is used by any player.
func (s *GameSessionInput) isDpadTouched(port uint, axis uint) (shift int16) {
	return int16(atomic.LoadInt32(&s[port].axes[axis]))
}
