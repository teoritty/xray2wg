package service

import "sync"

// RunSet is a concurrent-safe set of int64 IDs (e.g. tunnels considered running in memory).
type RunSet struct {
	mu sync.RWMutex
	m  map[int64]struct{}
}

// NewRunSet returns an empty RunSet.
func NewRunSet() *RunSet {
	return &RunSet{m: make(map[int64]struct{})}
}

// Mark adds id to the set.
func (s *RunSet) Mark(id int64) {
	s.mu.Lock()
	s.m[id] = struct{}{}
	s.mu.Unlock()
}

// Clear removes id from the set.
func (s *RunSet) Clear(id int64) {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
}

// Contains reports whether id is in the set.
func (s *RunSet) Contains(id int64) bool {
	s.mu.RLock()
	_, ok := s.m[id]
	s.mu.RUnlock()
	return ok
}

// Len returns the number of ids in the set.
func (s *RunSet) Len() int {
	s.mu.RLock()
	n := len(s.m)
	s.mu.RUnlock()
	return n
}

// IDs returns a copy of all ids (safe to mutate the slice; does not affect the set).
func (s *RunSet) IDs() []int64 {
	s.mu.RLock()
	out := make([]int64, 0, len(s.m))
	for id := range s.m {
		out = append(out, id)
	}
	s.mu.RUnlock()
	return out
}

// Drain removes and returns all ids, leaving the set empty.
func (s *RunSet) Drain() []int64 {
	s.mu.Lock()
	out := make([]int64, 0, len(s.m))
	for id := range s.m {
		out = append(out, id)
	}
	s.m = make(map[int64]struct{})
	s.mu.Unlock()
	return out
}
