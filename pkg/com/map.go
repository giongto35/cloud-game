package com

import (
	"fmt"
	"iter"
	"sync"
)

// Map defines a concurrent-safe map structure.
// Keep in mind that the underlying map structure will grow indefinitely.
type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

func (m *Map[K, _]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

func (m *Map[K, _]) Has(key K) bool {
	m.mu.RLock()
	_, ok := m.m[key]
	m.mu.RUnlock()
	return ok
}

// Get returns the value and exists flag (standard map comma-ok idiom).
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.m[key]
	return val, ok
}

func (m *Map[K, V]) Find(key K) V {
	v, _ := m.Get(key)
	return v
}

func (m *Map[K, V]) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fmt.Sprintf("%v", m.m)
}

// FindBy searches for the first value satisfying the predicate.
// Note: This holds a Read Lock during iteration.
func (m *Map[K, V]) FindBy(predicate func(v V) bool) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.m {
		if predicate(v) {
			return v, true
		}
	}
	var zero V
	return zero, false
}

// Put sets the value and returns true if the key already existed.
func (m *Map[K, V]) Put(key K, v V) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.m == nil {
		m.m = make(map[K]V)
	}

	_, exists := m.m[key]
	m.m[key] = v
	return exists
}

func (m *Map[K, V]) Remove(key K) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

// Pop returns the value and removes it from the map.
// Returns zero value if not found.
func (m *Map[K, V]) Pop(key K) V {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.m[key]
	if ok {
		delete(m.m, key)
	}
	return val
}

// RemoveL removes the key and returns the new length of the map.
func (m *Map[K, _]) RemoveL(key K) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
	return len(m.m)
}

// Clear empties the map.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	m.m = make(map[K]V)
	m.mu.Unlock()
}

// Values returns an iterator for values only.
//
// Usage: for k, v := range m.Values() { ... }
//
// Warning: This holds a Read Lock (RLock) during iteration.
// Do not call Put/Remove on this map inside the loop (Deadlock).
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		for _, v := range m.m {
			if !yield(v) {
				return
			}
		}
	}
}
