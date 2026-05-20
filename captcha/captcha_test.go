package captcha

import (
	"bytes"
	"context"
	"image/png"
	"testing"
	"time"
)

func TestGenerateAndVerify(t *testing.T) {
	c := New(Options{Length: 4, TTL: time.Minute})
	ctx := context.Background()

	id, imgData, err := c.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if id == "" {
		t.Error("empty id")
	}
	if len(imgData) == 0 {
		t.Error("empty png")
	}
	// Decode PNG to verify it's a valid image.
	if _, err := png.Decode(bytes.NewReader(imgData)); err != nil {
		t.Errorf("png.Decode: %v", err)
	}
}

func TestVerify_WrongAnswer(t *testing.T) {
	c := New(Options{})
	ctx := context.Background()
	id, _, _ := c.Generate(ctx)
	ok, err := c.Verify(ctx, id, "definitely-wrong-answer")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("verify should fail for wrong answer")
	}
}

func TestVerify_SingleUse(t *testing.T) {
	c := New(Options{})
	ctx := context.Background()
	id, _, _ := c.Generate(ctx)

	// We don't know the answer (random), but Verify must consume the entry
	// regardless of correctness so the next call returns false.
	_, _ = c.Verify(ctx, id, "guess")
	ok, err := c.Verify(ctx, id, "guess-again")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("second Verify must not succeed")
	}
}

func TestVerify_UnknownID(t *testing.T) {
	c := New(Options{})
	ok, err := c.Verify(context.Background(), "nonexistent", "x")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("unknown id should not verify")
	}
}

func TestCaseSensitivity(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Put(context.Background(), "ID1", "ABC", time.Minute)
	c := New(Options{Store: store, CaseSensitive: false})
	ok, _ := c.Verify(context.Background(), "ID1", "abc")
	if !ok {
		t.Error("case-insensitive should match")
	}

	store2 := NewMemoryStore()
	_ = store2.Put(context.Background(), "ID2", "ABC", time.Minute)
	c2 := New(Options{Store: store2, CaseSensitive: true})
	ok, _ = c2.Verify(context.Background(), "ID2", "abc")
	if ok {
		t.Error("case-sensitive should NOT match different case")
	}
}

func TestTTLExpiry(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Put(context.Background(), "X", "answer", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	v, ok, _ := store.Take(context.Background(), "X")
	if ok || v != "" {
		t.Errorf("expired entry should not return; got %q ok=%v", v, ok)
	}
}

func TestMemoryStore_SweepEvictsExpired(t *testing.T) {
	s := NewMemoryStoreWithInterval(20 * time.Millisecond)
	defer s.Close()

	_ = s.Put(context.Background(), "live", "x", time.Minute)
	_ = s.Put(context.Background(), "dead-1", "x", 5*time.Millisecond)
	_ = s.Put(context.Background(), "dead-2", "x", 5*time.Millisecond)

	if s.Len() != 3 {
		t.Fatalf("initial Len = %d", s.Len())
	}
	// Wait for the sweeper to fire at least once after the dead entries expire.
	time.Sleep(80 * time.Millisecond)
	if got := s.Len(); got != 1 {
		t.Errorf("after sweep, Len = %d; expected only the live entry to remain", got)
	}
}

func TestMemoryStore_CloseIdempotent(t *testing.T) {
	s := NewMemoryStoreWithInterval(5 * time.Millisecond)
	if err := s.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMemoryStore_Take_Atomic(t *testing.T) {
	s := NewMemoryStore()
	_ = s.Put(context.Background(), "k", "v", time.Minute)
	got1, ok1, _ := s.Take(context.Background(), "k")
	got2, ok2, _ := s.Take(context.Background(), "k")
	if !ok1 || got1 != "v" {
		t.Errorf("first Take: %q ok=%v", got1, ok1)
	}
	if ok2 || got2 != "" {
		t.Errorf("second Take should be gone: %q ok=%v", got2, ok2)
	}
}

func TestRandomAnswer_Length(t *testing.T) {
	c := New(Options{Length: 7, Charset: "ABC"})
	a := c.randomAnswer()
	if len(a) != 7 {
		t.Errorf("len = %d", len(a))
	}
	for _, r := range a {
		if r != 'A' && r != 'B' && r != 'C' {
			t.Errorf("unexpected rune %q in answer %q", r, a)
		}
	}
}

func TestRandomID_UniqueAndHexShape(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id, err := randomID()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Errorf("duplicate id %s", id)
		}
		seen[id] = true
		if len(id) != 24 {
			t.Errorf("expected 24 hex chars, got %q", id)
		}
	}
}
