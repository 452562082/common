package idgen

import (
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestUUID(t *testing.T) {
	a, b := UUID(), UUID()
	if a == b {
		t.Errorf("UUID collision: %s", a)
	}
	if len(a) != 36 || strings.Count(a, "-") != 4 {
		t.Errorf("unexpected format: %s", a)
	}
}

func TestUUIDv7_IsTimeOrdered(t *testing.T) {
	ids := make([]string, 50)
	for i := range ids {
		ids[i] = UUIDv7()
		time.Sleep(time.Millisecond)
	}
	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Strings(sorted)

	for i := range ids {
		if ids[i] != sorted[i] {
			t.Fatalf("UUID v7 not time-ordered at index %d:\ngot:    %s\nsorted: %s", i, ids[i], sorted[i])
		}
	}
}

func TestSnowflake_NodeRange(t *testing.T) {
	if _, err := NewSnowflake(-1); err == nil {
		t.Error("expected error for nodeID -1")
	}
	if _, err := NewSnowflake(1024); err == nil {
		t.Error("expected error for nodeID 1024")
	}
}

func TestSnowflake_MonotonicSingleThread(t *testing.T) {
	s, err := NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	prev := s.Next()
	for i := 0; i < 1000; i++ {
		next := s.Next()
		if next <= prev {
			t.Fatalf("non-monotonic: prev=%d next=%d", prev, next)
		}
		prev = next
	}
}

func TestSnowflake_ConcurrentNoDuplicates(t *testing.T) {
	s, err := NewSnowflake(2)
	if err != nil {
		t.Fatal(err)
	}
	const goroutines = 8
	const perG = 1000

	var mu sync.Mutex
	seen := make(map[int64]struct{}, goroutines*perG)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				id := s.Next()
				mu.Lock()
				if _, dup := seen[id]; dup {
					t.Errorf("duplicate id %d", id)
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if got := len(seen); got != goroutines*perG {
		t.Errorf("got %d unique ids, want %d", got, goroutines*perG)
	}
}

func TestSnowflake_StringFormats(t *testing.T) {
	s, _ := NewSnowflake(3)
	if id := s.NextString(); id == "" {
		t.Error("NextString empty")
	}
	if id := s.NextBase58(); id == "" {
		t.Error("NextBase58 empty")
	}
}
