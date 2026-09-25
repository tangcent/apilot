package collector

import "sync"

// UnresolvedReporter is an optional interface implemented by collectors whose
// type resolvers can report type names they failed to expand.
//
// A resolver that cannot resolve a type falls back to an opaque single-value
// model, which is structurally identical to a resolved primitive. Collectors
// implementing this interface let callers (e.g. the engine) tell the two
// apart and surface a diagnostic instead of silently dropping fields.
//
// It is intentionally not part of Collector: subprocess plugins cannot report
// across the process boundary, so the engine must type-assert.
type UnresolvedReporter interface {
	// Unresolved returns the unresolved type names mapped to how many times
	// each one was encountered during the last Collect call.
	Unresolved() map[string]int
}

// UnresolvedSet accumulates unresolved type names across concurrent parsers.
// A nil *UnresolvedSet is usable: Record is a no-op and Counts returns nil,
// so parsers can accept it as an optional parameter.
type UnresolvedSet struct {
	mu     sync.Mutex
	counts map[string]int
}

// NewUnresolvedSet returns an empty, ready-to-use UnresolvedSet.
func NewUnresolvedSet() *UnresolvedSet {
	return &UnresolvedSet{counts: make(map[string]int)}
}

// Record increments the occurrence count for typeName. Empty names are ignored:
// they mean "no type was written at all", which is not a resolution failure.
func (s *UnresolvedSet) Record(typeName string) {
	if s == nil || typeName == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts == nil {
		s.counts = make(map[string]int)
	}
	s.counts[typeName]++
}

// Counts returns a copy of the accumulated counts, safe to read after Record.
func (s *UnresolvedSet) Counts() map[string]int {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.counts))
	for name, n := range s.counts {
		out[name] = n
	}
	return out
}
