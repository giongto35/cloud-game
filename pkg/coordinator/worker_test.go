package coordinator

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestSlotted(t *testing.T) {
	t.Run("UnReserve", func(t *testing.T) {
		t.Run("BasicDecrement", testUnReserveBasic)
		t.Run("PreventUnderflow", testUnReserveUnderflow)
		t.Run("ConcurrentDecrement", testUnReserveConcurrent)
	})

	t.Run("TryReserve", func(t *testing.T) {
		t.Run("SuccessWhenZero", testTryReserveSuccess)
		t.Run("FailWhenNonZero", testTryReserveFailure)
		t.Run("ConcurrentReservations", testTryReserveConcurrent)
	})

	t.Run("Integration", func(t *testing.T) {
		t.Run("ReserveUnreserveFlow", testReserveUnreserveFlow)
		t.Run("FreeSlots", testFreeSlots)
		t.Run("HasSlot", testHasSlot)
	})
}

func testUnReserveBasic(t *testing.T) {
	t.Parallel()
	var s slotted

	// Initial state
	if atomic.LoadInt32((*int32)(&s)) != 0 {
		t.Fatal("initial state not zero")
	}

	// Test normal decrement
	s.TryReserve() // 0 -> 1
	s.UnReserve()
	if atomic.LoadInt32((*int32)(&s)) != 0 {
		t.Error("failed to decrement to zero")
	}

	// Test multiple decrements
	s.TryReserve() // 0 -> 1
	s.TryReserve() // 1 -> 2
	s.UnReserve()
	s.UnReserve()
	if atomic.LoadInt32((*int32)(&s)) != 0 {
		t.Error("failed to decrement multiple times")
	}
}

func testUnReserveUnderflow(t *testing.T) {
	t.Parallel()
	var s slotted

	t.Run("PreventNewUnderflow", func(t *testing.T) {
		s.UnReserve() // Start at 0
		if atomic.LoadInt32((*int32)(&s)) != 0 {
			t.Error("should remain at 0 when unreserving from 0")
		}
	})

	t.Run("FixExistingNegative", func(t *testing.T) {
		atomic.StoreInt32((*int32)(&s), -5)
		s.UnReserve()
		if current := atomic.LoadInt32((*int32)(&s)); current != 0 {
			t.Errorf("should fix negative value to 0, got %d", current)
		}
	})
}

func testUnReserveConcurrent(t *testing.T) {
	t.Parallel()

	var s slotted
	const workers = 100
	var wg sync.WaitGroup

	atomic.StoreInt32((*int32)(&s), int32(workers))
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			s.UnReserve()
		}()
	}

	wg.Wait()

	if current := atomic.LoadInt32((*int32)(&s)); current != 0 {
		t.Errorf("unexpected final value: %d (want 0)", current)
	}
}

func testTryReserveSuccess(t *testing.T) {
	t.Parallel()
	var s slotted

	if !s.TryReserve() {
		t.Error("should succeed when zero")
	}
	if atomic.LoadInt32((*int32)(&s)) != 1 {
		t.Error("failed to increment")
	}
}

func testTryReserveFailure(t *testing.T) {
	t.Parallel()
	var s slotted

	atomic.StoreInt32((*int32)(&s), 1)
	if s.TryReserve() {
		t.Error("should fail when non-zero")
	}
}

func testTryReserveConcurrent(t *testing.T) {
	t.Parallel()
	var s slotted
	const workers = 100
	var success int32
	var wg sync.WaitGroup

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			if s.TryReserve() {
				atomic.AddInt32(&success, 1)
			}
		}()
	}

	wg.Wait()

	if success != 1 {
		t.Errorf("unexpected success count: %d (want 1)", success)
	}
	if atomic.LoadInt32((*int32)(&s)) != 1 {
		t.Error("counter not properly incremented")
	}
}

func testReserveUnreserveFlow(t *testing.T) {
	t.Parallel()
	var s slotted

	// Successful reservation
	if !s.TryReserve() {
		t.Fatal("failed initial reservation")
	}

	// Second reservation should fail
	if s.TryReserve() {
		t.Error("unexpected successful second reservation")
	}

	// Unreserve and try again
	s.UnReserve()
	if !s.TryReserve() {
		t.Error("failed reservation after unreserve")
	}
}

func testFreeSlots(t *testing.T) {
	t.Parallel()
	var s slotted

	// Set to arbitrary value
	atomic.StoreInt32((*int32)(&s), 5)
	s.FreeSlots()
	if atomic.LoadInt32((*int32)(&s)) != 0 {
		t.Error("FreeSlots failed to reset counter")
	}
}

func testHasSlot(t *testing.T) {
	t.Parallel()
	var s slotted

	if !s.HasSlot() {
		t.Error("should have slot when zero")
	}

	s.TryReserve()
	if s.HasSlot() {
		t.Error("shouldn't have slot when reserved")
	}
}
