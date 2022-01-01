package client

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
func (t *testClient) Close()          {}
func (t *testClient) change(n int)    { atomic.AddInt32(&t.c, int32(n)) }

func TestPointerValue(t *testing.T) {
	m := NewNetMap()
	c := testClient{id: 1}
	m.Add(&c)
	fc, _ := m.FindBy(func(c NetClient) bool {
		cc := c.(*testClient)
		if cc.id == 1 {
			return true
		}
		return false
	})
	c.change(100)
	fc2, _ := m.Find(fc.(*testClient).Id().String())

	expected := c.c == fc.(*testClient).c && c.c == fc2.(*testClient).c
	if !expected {
		t.Errorf("not expected change, o: %v != %v != %v", c.c, fc.(*testClient).c, fc2.(*testClient).c)
	}
}
