package service

import (
	"runtime"
	"sort"
	"sync"
	"testing"
)

func TestRunSetConcurrentMarkClearContains(t *testing.T) {
	s := NewRunSet()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(k int64) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				s.Mark(k)
				_ = s.Contains(k)
				s.Clear(k)
				_ = s.Len()
				_ = s.IDs()
			}
		}(int64(i))
	}
	wg.Wait()
}

func TestRunSetDrainClears(t *testing.T) {
	s := NewRunSet()
	s.Mark(1)
	s.Mark(2)
	got := s.Drain()
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if s.Len() != 0 {
		t.Fatal("expected empty after drain")
	}
}

// TestRunSetDrainReturnsAllAndClears is the production-hardening contract: Drain is atomic
// (lock + copy + clear) and safe under concurrent Mark/Clear/Drain.
func TestRunSetDrainReturnsAllAndClears(t *testing.T) {
	t.Run("two_ids_snapshot", func(t *testing.T) {
		s := NewRunSet()
		s.Mark(1)
		s.Mark(2)
		got := s.Drain()
		if len(got) != 2 {
			t.Fatalf("len %d", len(got))
		}
		if s.Len() != 0 {
			t.Fatal("expected empty after drain")
		}
	})
	t.Run("concurrent_drain_atomicity", func(t *testing.T) {
		runConcurrentDrainExactlyOnceNonEmpty(t)
	})
	t.Run("drain_low_ids_under_disjoint_high_id_churn", func(t *testing.T) {
		runDrainLowIDsUnderDisjointHighIDChurn(t)
	})
}

// runConcurrentDrainExactlyOnceNonEmpty verifies Drain's critical section: only one concurrent
// Drain observes the pre-populated map; all others see empty. Every id appears in exactly one slice.
func runConcurrentDrainExactlyOnceNonEmpty(t *testing.T) {
	t.Helper()
	const nIDs = 100
	const nDrainers = 32
	s := NewRunSet()
	for id := int64(1); id <= nIDs; id++ {
		s.Mark(id)
	}
	var wg sync.WaitGroup
	results := make(chan []int64, nDrainers)
	for i := 0; i < nDrainers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- s.Drain()
		}()
	}
	wg.Wait()
	close(results)
	var nonEmpty int
	seen := make(map[int64]struct{})
	var total int
	for ids := range results {
		if len(ids) > 0 {
			nonEmpty++
		}
		total += len(ids)
		for _, id := range ids {
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate id %d returned by concurrent Drain", id)
			}
			seen[id] = struct{}{}
		}
	}
	if nonEmpty != 1 {
		t.Fatalf("expected exactly one non-empty Drain, got %d (atomic drain+clear)", nonEmpty)
	}
	if total != nIDs {
		t.Fatalf("expected %d ids drained in total, got %d (lost or duplicated work)", nIDs, total)
	}
	if s.Len() != 0 {
		t.Fatalf("set must be empty after all drains, Len=%d", s.Len())
	}
}

// runDrainLowIDsUnderDisjointHighIDChurn hammers Mark/Clear on ids >= 10000 while the test
// repeatedly Marks low ids 0..19 and Drains. Low ids must never be duplicated or lost inside
// a single Drain batch (workers never touch the low range).
func runDrainLowIDsUnderDisjointHighIDChurn(t *testing.T) {
	t.Helper()
	const lowN = 20
	const highBase = int64(10000)
	s := NewRunSet()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			id := highBase + int64(g)
			for {
				select {
				case <-stop:
					return
				default:
					s.Mark(id)
					runtime.Gosched()
					s.Clear(id)
					runtime.Gosched()
				}
			}
		}(g)
	}
	for round := 0; round < 200; round++ {
		for id := int64(0); id < lowN; id++ {
			s.Mark(id)
		}
		got := s.Drain()
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		seen := make(map[int64]int, len(got))
		for _, id := range got {
			seen[id]++
			if id >= 0 && id < lowN && seen[id] > 1 {
				t.Fatalf("round %d: duplicate low id %d in one Drain batch", round, id)
			}
		}
		for id := int64(0); id < lowN; id++ {
			if seen[id] != 1 {
				t.Fatalf("round %d: low id %d must appear exactly once in Drain output, counts=%v", round, id, seen)
			}
		}
	}
	close(stop)
	wg.Wait()
}

func TestRunSetIDsIsCopy(t *testing.T) {
	s := NewRunSet()
	s.Mark(7)
	a := s.IDs()
	a[0] = 999
	b := s.IDs()
	if len(b) != 1 || b[0] != 7 {
		t.Fatalf("internal set mutated: %v", b)
	}
}

func TestTunnelServiceRunningSetConcurrent(t *testing.T) {
	svc := &TunnelService{running: NewRunSet()}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		id := int64(i % 5)
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.running.Mark(id)
			_ = svc.IsRunning(id)
			svc.running.Clear(id)
		}()
	}
	wg.Wait()
}
