package com

import (
	"errors"
	"sync"
)

// Map defines a concurrent-safe map structure.
// Keep in mind that underlying map structure will grow indefinitely.
type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.Mutex
}

var ErrNotFound = errors.New("not found")

func (m *Map[K, _]) Has(key K) bool { _, err := m.Find(key); return err == nil }
func (m *Map[_, _]) IsEmpty() bool  { m.mu.Lock(); defer m.mu.Unlock(); return len(m.m) == 0 }
func (m *Map[K, T]) List() map[K]T  { return m.m }
func (m *Map[K, T]) Pop(key K) T {
	m.mu.Lock()
	v := m.m[key]
	delete(m.m, key)
	m.mu.Unlock()
	return v
}
func (m *Map[K, T]) Put(key K, v T) { m.mu.Lock(); m.m[key] = v; m.mu.Unlock() }
func (m *Map[K, _]) Remove(key K)   { m.mu.Lock(); delete(m.m, key); m.mu.Unlock() }

// Find searches for the first match, returns ErrNotFound otherwise.
func (m *Map[K, T]) Find(key K) (v T, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if vv, ok := m.m[key]; ok {
		return vv, nil
	}
	return v, ErrNotFound
}

// FindBy searches the first key-value with the provided predicate function.
func (m *Map[K, T]) FindBy(fn func(v T) bool) (v T, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, vv := range m.m {
		if fn(vv) {
			return vv, nil
		}
	}
	return v, ErrNotFound
}

// ForEach processes every element with the provided callback function.
func (m *Map[K, T]) ForEach(fn func(v T)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.m {
		fn(v)
	}
}
