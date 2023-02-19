package com

import (
	"errors"
	"sync"
)

// NetMap defines a thread-safe NetClient list.
type NetMap[T NetClient] struct {
	m  map[string]T
	mu sync.Mutex
}

// ErrNotFound is returned by NetMap when some value is not present.
var ErrNotFound = errors.New("not found")

func NewNetMap[T NetClient]() NetMap[T] { return NetMap[T]{m: make(map[string]T, 10)} }

// Add adds a new NetClient value with its id value as the key.
func (m *NetMap[T]) Add(client T) { m.Put(client.Id().String(), client) }

// Put adds a new NetClient value with a custom key value.
func (m *NetMap[T]) Put(key string, client T) {
	m.mu.Lock()
	m.m[key] = client
	m.mu.Unlock()
}

// Remove removes NetClient from the map if present.
func (m *NetMap[T]) Remove(client T) { m.RemoveByKey(client.Id().String()) }

func (m *NetMap[T]) RemoveDisconnect(client T) {
	client.Disconnect()
	m.Remove(client)
}

// RemoveByKey removes NetClient from the map by a specified key value.
func (m *NetMap[T]) RemoveByKey(key string) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

// RemoveAll removes all occurrences of specified NetClient.
func (m *NetMap[T]) RemoveAll(client T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, c := range m.m {
		if c.Id() == client.Id() {
			delete(m.m, k)
		}
	}
}

func (m *NetMap[T]) IsEmpty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.m) == 0
}

// List returns the current NetClient map.
func (m *NetMap[T]) List() map[string]T { return m.m }

func (m *NetMap[T]) Has(key string) bool {
	_, err := m.Find(key)
	return err == nil
}

// Find searches the first NetClient by a specified key value.
func (m *NetMap[T]) Find(key string) (client T, err error) {
	if key == "" {
		return client, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.m[key]; ok {
		return c, nil
	}
	return client, ErrNotFound
}

// FindBy searches the first NetClient with the provided predicate function.
func (m *NetMap[T]) FindBy(fn func(c T) bool) (client T, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.m {
		if fn(w) {
			return w, nil
		}
	}
	return client, ErrNotFound
}

// ForEach processes every NetClient with the provided callback function.
func (m *NetMap[T]) ForEach(fn func(c T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.m {
		fn(w)
	}
}
