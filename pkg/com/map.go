package com

import (
	"errors"
	"sync"
)

// NetMap defines a thread-safe NetClient list.
type NetMap[K comparable, T NetClient[K]] struct {
	m  map[K]T
	mu sync.Mutex
}

// ErrNotFound is returned by NetMap when some value is not present.
var ErrNotFound = errors.New("not found")

func NewNetMap[K comparable, T NetClient[K]]() NetMap[K, T] {
	return NetMap[K, T]{m: make(map[K]T, 10)}
}

// Add adds a new NetClient value with its id value as the key.
func (m *NetMap[K, T]) Add(client T) { m.Put(client.Id(), client) }

// Put adds a new NetClient value with a custom key value.
func (m *NetMap[K, T]) Put(key K, client T) {
	m.mu.Lock()
	m.m[key] = client
	m.mu.Unlock()
}

// Remove removes NetClient from the map if present.
func (m *NetMap[K, T]) Remove(client T) { m.RemoveByKey(client.Id()) }

func (m *NetMap[K, T]) RemoveDisconnect(client T) {
	client.Disconnect()
	m.Remove(client)
}

// RemoveByKey removes NetClient from the map by a specified key value.
func (m *NetMap[K, T]) RemoveByKey(key K) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

// RemoveAll removes all occurrences of specified NetClient.
func (m *NetMap[K, T]) RemoveAll(client T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, c := range m.m {
		if c.Id() == client.Id() {
			delete(m.m, k)
		}
	}
}

func (m *NetMap[K, T]) IsEmpty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.m) == 0
}

// List returns the current NetClient map.
func (m *NetMap[K, T]) List() map[K]T { return m.m }

func (m *NetMap[K, T]) Has(key K) bool {
	_, err := m.Find(key)
	return err == nil
}

// Find searches the first NetClient by a specified key value.
func (m *NetMap[K, T]) Find(key K) (client T, err error) {
	var empty K
	if key == empty {
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
func (m *NetMap[K, T]) FindBy(fn func(c T) bool) (client T, err error) {
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
func (m *NetMap[K, T]) ForEach(fn func(c T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.m {
		fn(w)
	}
}
