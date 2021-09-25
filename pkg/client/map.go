package client

import (
	"errors"
	"sync"
)

// NetMap defines a thread-safe NetClient list.
type NetMap struct {
	sync.Mutex
	m map[string]NetClient
}

// ErrNotFound is returned by NetMap when some value is not present.
var ErrNotFound = errors.New("not found")

func NewNetMap(m map[string]NetClient) NetMap {
	return NetMap{m: m}
}

// Add adds a new NetClient value with its id value as the key.
func (m *NetMap) Add(client NetClient) { m.Put(string(client.Id()), client) }

// Put adds a new NetClient value with a custom key value.
func (m *NetMap) Put(key string, client NetClient) {
	m.Lock()
	m.m[key] = client
	m.Unlock()
}

// Remove removes NetClient from the map if present.
func (m *NetMap) Remove(client NetClient) { m.RemoveByKey(string(client.Id())) }

// RemoveByKey removes NetClient from the map by a specified key value.
func (m *NetMap) RemoveByKey(key string) {
	m.Lock()
	delete(m.m, key)
	m.Unlock()
}

// RemoveAll removes specified NetClient from the map
// no matter with how many custom keys it was stored.
func (m *NetMap) RemoveAll(client NetClient) {
	m.Lock()
	defer m.Unlock()
	for k, cur := range m.m {
		if string(cur.Id()) == string(client.Id()) {
			delete(m.m, k)
		}
	}
}

// List returns the current NetClient map.
func (m *NetMap) List() map[string]NetClient { return m.m }

// Find searches the first NetClient by a specified key value.
func (m *NetMap) Find(key string) (client NetClient, err error) {
	if key == "" {
		return client, ErrNotFound
	}
	m.Lock()
	defer m.Unlock()
	if c, ok := m.m[key]; ok {
		return c, nil
	}
	return client, ErrNotFound
}

// FindBy searches the first NetClient with the provided predicate function.
func (m *NetMap) FindBy(fn func(_ NetClient) bool) (client NetClient, err error) {
	m.Lock()
	defer m.Unlock()
	for _, w := range m.m {
		if fn(w) {
			return w, nil
		}
	}
	return client, ErrNotFound
}

// ForEach processes every NetClient with the provided callback function.
func (m *NetMap) ForEach(fn func(_ NetClient)) {
	m.Lock()
	defer m.Unlock()
	for _, w := range m.m {
		fn(w)
	}
}
