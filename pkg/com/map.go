package com

import (
	"errors"
	"sync"
)

// Map defines a concurrent-safe map structure.
type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.Mutex
}

var ErrNotFound = errors.New("not found")

func (m *Map[K, _]) Has(key K) bool      { _, err := m.Find(key); return err == nil }
func (m *Map[_, _]) IsEmpty() bool       { m.mu.Lock(); defer m.mu.Unlock(); return len(m.m) == 0 }
func (m *Map[K, T]) List() map[K]T       { return m.m }
func (m *Map[K, T]) Pop(key K) T         { m.mu.Lock(); defer m.mu.Unlock(); return m.m[key] }
func (m *Map[K, T]) Put(key K, client T) { m.mu.Lock(); m.m[key] = client; m.mu.Unlock() }
func (m *Map[K, _]) RemoveByKey(key K)   { m.mu.Lock(); delete(m.m, key); m.mu.Unlock() }

// Find searches for the first match by a specified key value,
// returns ErrNotFound otherwise.
func (m *Map[K, T]) Find(key K) (client T, err error) {
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

// FindBy searches the first key-value with the provided predicate function.
func (m *Map[K, T]) FindBy(fn func(c T) bool) (client T, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.m {
		if fn(w) {
			return w, nil
		}
	}
	return client, ErrNotFound
}

// ForEach processes every element with the provided callback function.
func (m *Map[K, T]) ForEach(fn func(c T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.m {
		fn(w)
	}
}
