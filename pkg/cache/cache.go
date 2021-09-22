package cache

import (
	"errors"
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/client"
)

type Cache struct {
	mu   sync.Mutex
	data map[string]client.NetClient
}

func New(storage map[string]client.NetClient) Cache {
	return Cache{data: storage}
}

func (c *Cache) Add(id string, client client.NetClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[id] = client
}

func (c *Cache) Remove(id string) {
	c.mu.Lock()
	delete(c.data, id)
	c.mu.Unlock()
}

func (c *Cache) RemoveAllWithId(id string) {
	for k, server := range c.data {
		if string(server.Id()) == id {
			c.Remove(k)
			server.Printf("Room has been destroyed %s", id)
		}
	}
}

func (c *Cache) List() map[string]client.NetClient {
	return c.data
}

func (c *Cache) Find(id string) (cl client.NetClient, err error) {
	if id == "" {
		return cl, errors.New("not found")
	}
	if c, ok := c.data[id]; ok {
		return c, nil
	}
	return cl, errors.New("not found")
}

func (c *Cache) FindBy(fn func(cl client.NetClient) bool) (cl client.NetClient, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range c.data {
		if fn(w) {
			return w, nil
		}
	}
	return cl, errors.New("not found")
}

func (c *Cache) ForEach(fn func(cl client.NetClient)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range c.data {
		fn(w)
	}
}
