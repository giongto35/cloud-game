package coordinator

import (
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/coordinator/user"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

// Crowd denotes some abstraction over list of eager people.
type Crowd struct {
	mu    sync.Mutex
	users map[network.Uid]*user.User
}

func NewCrowd() Crowd {
	return Crowd{
		users: map[network.Uid]*user.User{},
	}
}

func (c *Crowd) add(u *user.User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users[u.Id] = u
}

func (c *Crowd) finish(u *user.User) {
	if u == nil {
		return
	}
	c.mu.Lock()
	delete(c.users, u.Id)
	c.mu.Unlock()
	u.Clean()
}

func (c *Crowd) findById(id network.Uid) *user.User {
	c.mu.Lock()
	defer c.mu.Unlock()
	usr, ok := c.users[id]
	if ok {
		return usr
	}
	return nil
}
