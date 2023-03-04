package com

import "testing"

func TestMap_InGeneral(t *testing.T) {
	// map map
	m := Map[int, int]{m: make(map[int]int)}

	if !m.IsEmpty() {
		t.Errorf("should be empty, %v", m.m)
	}
	k := 0
	m.Put(k, 0)
	if m.IsEmpty() {
		t.Errorf("should not be empty, %v", m.m)
	}
	if !m.Has(k) {
		t.Errorf("should have the key %v, %v", k, m.m)
	}
	v, err := m.Find(k)
	if v != 0 && err != nil {
		t.Errorf("should have the key %v and no error, %v %v", k, err, m.m)
	}
	v, err = m.Find(k + 1)
	if err != ErrNotFound {
		t.Errorf("should not find anything, %v %v", err, m.m)
	}
	m.Put(1, 1)
	v, err = m.FindBy(func(v int) bool { return v == 1 })
	if v != 1 && err != nil {
		t.Errorf("should have the key %v and no error, %v %v", 1, err, m.m)
	}
	sum := 0
	m.ForEach(func(v int) { sum += v })
	if sum != 1 {
		t.Errorf("shoud have exact sum of 1, but have %v", sum)
	}
	m.Remove(1)
	if !m.Has(0) || len(m.List()) > 1 {
		t.Errorf("should remove only one element, but has %v", m.m)
	}
	m.Put(3, 3)
	v = m.Pop(3)
	if v != 3 {
		t.Errorf("should have value %v, but has %v %v", 3, v, m.m)
	}
	m.Remove(3)
	m.Remove(0)
	if len(m.List()) != 0 {
		t.Errorf("should be completely empty, but %v", m.m)
	}
}

func TestMap_Concurrency(t *testing.T) {
	m := Map[int, int]{m: make(map[int]int)}
	for i := 0; i < 100; i++ {
		i := i
		go m.Put(i, i)
		go m.Has(i)
		go m.Pop(i)
	}
}
