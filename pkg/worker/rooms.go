package worker

import "sync"

type Rooms struct {
	mu    sync.Mutex
	store map[string]*Room
}

func NewRooms() Rooms { return Rooms{store: make(map[string]*Room, 10)} }

func (r *Rooms) Add(room *Room) {
	r.mu.Lock()
	r.store[room.ID] = room
	r.mu.Unlock()
}

func (r *Rooms) Get(id string) *Room {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.store[id]
}

func (r *Rooms) Remove(id string) {
	r.mu.Lock()
	delete(r.store, id)
	r.mu.Unlock()
}

func (r *Rooms) Close() {
	for _, r := range r.store {
		r.Close()
	}
}
