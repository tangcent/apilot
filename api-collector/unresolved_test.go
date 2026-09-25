package collector

import (
	"sync"
	"testing"
)

func TestUnresolvedSet_RecordCountsOccurrences(t *testing.T) {
	s := NewUnresolvedSet()

	s.Record("PageResult")
	s.Record("PageResult")
	s.Record("ApiResponse")

	counts := s.Counts()
	if counts["PageResult"] != 2 {
		t.Errorf("PageResult count = %d, want 2", counts["PageResult"])
	}
	if counts["ApiResponse"] != 1 {
		t.Errorf("ApiResponse count = %d, want 1", counts["ApiResponse"])
	}
}

func TestUnresolvedSet_EmptyNameIsNotUnresolved(t *testing.T) {
	s := NewUnresolvedSet()

	s.Record("")

	if len(s.Counts()) != 0 {
		t.Errorf("empty type name must not be recorded, got %v", s.Counts())
	}
}

func TestUnresolvedSet_NilSinkIsUsable(t *testing.T) {
	var s *UnresolvedSet

	s.Record("PageResult") // must not panic

	if len(s.Counts()) != 0 {
		t.Errorf("nil set Counts() = %v, want empty", s.Counts())
	}
}

func TestUnresolvedSet_CountsReturnsCopy(t *testing.T) {
	s := NewUnresolvedSet()
	s.Record("PageResult")

	counts := s.Counts()
	counts["PageResult"] = 99
	counts["Injected"] = 1

	if s.Counts()["PageResult"] != 1 {
		t.Errorf("mutating Counts() result changed the set: %v", s.Counts())
	}
	if _, ok := s.Counts()["Injected"]; ok {
		t.Errorf("mutating Counts() result changed the set: %v", s.Counts())
	}
}

func TestUnresolvedSet_ConcurrentRecord(t *testing.T) {
	s := NewUnresolvedSet()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Record("PageResult")
			s.Record("ApiResponse")
		}()
	}
	wg.Wait()

	counts := s.Counts()
	if counts["PageResult"] != 50 {
		t.Errorf("PageResult count = %d, want 50", counts["PageResult"])
	}
	if counts["ApiResponse"] != 50 {
		t.Errorf("ApiResponse count = %d, want 50", counts["ApiResponse"])
	}
}
