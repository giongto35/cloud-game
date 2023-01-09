package com

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type testClient struct {
	NetClient
	id int
	c  int32
}

func (t *testClient) Id() network.Uid { return network.Uid(fmt.Sprintf("%v", t.id)) }
func (t *testClient) change(n int)    { atomic.AddInt32(&t.c, int32(n)) }

func TestPointerValue(t *testing.T) {
	m := NewNetMap[*testClient]()
	c := testClient{id: 1}
	m.Add(&c)
	fc, _ := m.FindBy(func(c *testClient) bool { return c.id == 1 })
	c.change(100)
	fc2, _ := m.Find(fc.Id().String())

	expected := c.c == fc.c && c.c == fc2.c
	if !expected {
		t.Errorf("not expected change, o: %v != %v != %v", c.c, fc.c, fc2.c)
	}
}
