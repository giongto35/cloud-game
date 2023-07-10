package com

import "sync"

// Map defines a concurrent-safe map structure.
// Keep in mind that the underlying map structure will grow indefinitely.
type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.Mutex
}

func (m *Map[K, _]) Has(key K) bool { _, ok := m.Find(key); return ok }
func (m *Map[_, _]) Len() int       { m.mu.Lock(); defer m.mu.Unlock(); return len(m.m) }
func (m *Map[K, V]) Pop(key K) V {
	m.mu.Lock()
	v := m.m[key]
	delete(m.m, key)
	m.mu.Unlock()
	return v
}
func (m *Map[K, V]) Put(key K, v V) bool {
	m.mu.Lock()
	_, ok := m.m[key]
	m.m[key] = v
	m.mu.Unlock()
	return ok
}
func (m *Map[K, _]) Remove(key K) { m.mu.Lock(); delete(m.m, key); m.mu.Unlock() }

// Find returns the first value found and a boolean flag if its found or not.
func (m *Map[K, V]) Find(key K) (v V, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if vv, ok := m.m[key]; ok {
		return vv, true
	}
	return v, false
}

// FindBy searches the first key-value with the provided predicate function.
func (m *Map[K, V]) FindBy(fn func(v V) bool) (v V, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, vv := range m.m {
		if fn(vv) {
			return vv, true
		}
	}
	return v, false
}

// ForEach processes every element with the provided callback function.
func (m *Map[K, V]) ForEach(fn func(v V)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.m {
		fn(v)
	}
}
