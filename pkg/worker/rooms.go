package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
	"sync"
)

type Rooms struct {
	mu    sync.Mutex
	store map[string]*room.Room
}

func NewRooms() Rooms {
	return Rooms{store: make(map[string]*room.Room, 10)}
}

func (r *Rooms) Add(room *room.Room) {
	r.mu.Lock()
	r.store[room.ID] = room
	r.mu.Unlock()
}

func (r *Rooms) Get(id string) *room.Room {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.store[id]
}

// noSessions checks if a room has a running session.
// TODO: If we remove sessions from room anytime a session is closed,
// we can check if the sessions list is empty or not.
func (r *Rooms) noSessions(id string) bool {
	if id == "" {
		return true
	}
	rm := r.Get(id)
	if rm == nil {
		return true
	}
	return !rm.HasRunningSessions()
}

func (r *Rooms) Remove(id string) {
	r.mu.Lock()
	delete(r.store, id)
	r.mu.Unlock()
}

func (r *Rooms) CloseAll() {
	for _, r := range r.store {
		r.Close()
	}
}
