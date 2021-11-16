package worker

import "github.com/giongto35/cloud-game/v2/pkg/network"

type Router struct {
	rooms    Rooms
	sessions Sessions
}

func NewRouter() Router {
	return Router{
		rooms:    NewRooms(),
		sessions: NewSessions(),
	}
}

func (r *Router) AddRoom(room *Room) { r.rooms.Add(room) }

func (r *Router) AddUser(user *Session) { r.sessions.Add(user.id, user) }

func (r *Router) GetUser(uid network.Uid) *Session { return r.sessions.Get(uid) }

func (r *Router) GetRoom(id string) *Room { return r.rooms.Get(id) }

func (r *Router) RemoveRoom(room *Room) { r.rooms.Remove(room.ID) }

func (r *Router) RemoveUser(user *Session) { r.sessions.Remove(user) }

func (r *Router) IsRoomEmpty(id string) bool { return r.rooms.noSessions(id) }

func (r *Router) Close() {
	r.rooms.Close()
}
