package sequencer_test

import (
	"sync"
	"testing"

	"internal/shared/sequencer"
)

func key(venue, instrument, eventType string) sequencer.StreamKey {
	return sequencer.StreamKey{Venue: venue, Instrument: instrument, EventType: eventType}
}

func TestSequencer_FirstNextReturnsZero(t *testing.T) {
	s := sequencer.New()
	k := key("binance", "BTC/USDT-spot", "observation.trade")
	if got := s.Next(k); got != 0 {
		t.Fatalf("first Next = %d, want 0", got)
	}
}

func TestSequencer_MonotonicWithinKey(t *testing.T) {
	s := sequencer.New()
	k := key("binance", "BTC/USDT-spot", "observation.trade")
	prev := int64(-1)
	for i := 0; i < 1000; i++ {
		got := s.Next(k)
		if got <= prev {
			t.Fatalf("Next at iteration %d = %d, not strictly greater than previous %d", i, got, prev)
		}
		prev = got
	}
}

func TestSequencer_IndependentAcrossKeys(t *testing.T) {
	s := sequencer.New()
	k1 := key("binance", "BTC/USDT-spot", "observation.trade")
	k2 := key("binance", "ETH/USDT-spot", "observation.trade")
	k3 := key("binance", "BTC/USDT-spot", "observation.book.snapshot")

	// Interleave Next calls; each key should advance independently.
	for round := 0; round < 50; round++ {
		want := int64(round)
		if got := s.Next(k1); got != want {
			t.Fatalf("k1 round %d = %d, want %d", round, got, want)
		}
		if got := s.Next(k2); got != want {
			t.Fatalf("k2 round %d = %d, want %d", round, got, want)
		}
		if got := s.Next(k3); got != want {
			t.Fatalf("k3 round %d = %d, want %d", round, got, want)
		}
	}
}

func TestSequencer_RestoreResumesFromSnapshot(t *testing.T) {
	s := sequencer.New()
	k1 := key("binance", "BTC/USDT-spot", "observation.trade")
	k2 := key("binance", "ETH/USDT-spot", "observation.trade")
	for i := 0; i < 100; i++ {
		s.Next(k1)
	}
	for i := 0; i < 5; i++ {
		s.Next(k2)
	}
	snap := s.Snapshot()
	if snap[k1] != 99 {
		t.Fatalf("snapshot k1 = %d, want 99", snap[k1])
	}
	if snap[k2] != 4 {
		t.Fatalf("snapshot k2 = %d, want 4", snap[k2])
	}

	// New sequencer restores from snapshot and continues.
	s2 := sequencer.New()
	s2.Restore(snap)
	if got := s2.Next(k1); got != 100 {
		t.Fatalf("post-Restore k1 Next = %d, want 100", got)
	}
	if got := s2.Next(k2); got != 5 {
		t.Fatalf("post-Restore k2 Next = %d, want 5", got)
	}
}

func TestSequencer_RestoreIsReplaceNotMerge(t *testing.T) {
	s := sequencer.New()
	kA := key("binance", "AAA", "trade")
	kB := key("binance", "BBB", "trade")
	s.Next(kA) // 0
	s.Next(kA) // 1
	s.Next(kB) // 0

	// Restore a snapshot that mentions only kA at a high value.
	s.Restore(map[sequencer.StreamKey]int64{kA: 999})
	if got := s.Next(kA); got != 1000 {
		t.Fatalf("post-Restore kA Next = %d, want 1000", got)
	}
	// kB was dropped; first Next returns 0.
	if got := s.Next(kB); got != 0 {
		t.Fatalf("kB after Restore (dropped) Next = %d, want 0", got)
	}
}

func TestSequencer_SnapshotIsACopy(t *testing.T) {
	s := sequencer.New()
	k := key("binance", "BTC", "trade")
	s.Next(k) // 0
	s.Next(k) // 1
	snap := s.Snapshot()
	snap[k] = -999 // mutate the returned map
	if got, _ := s.Peek(k); got != 1 {
		t.Fatalf("Sequencer state mutated by snapshot caller: Peek = %d, want 1", got)
	}
}

func TestSequencer_SnapshotOmitsUnusedKeys(t *testing.T) {
	s := sequencer.New()
	if got := len(s.Snapshot()); got != 0 {
		t.Fatalf("fresh Sequencer snapshot has %d entries, want 0", got)
	}
	k := key("binance", "BTC", "trade")
	s.Next(k)
	snap := s.Snapshot()
	if _, exists := snap[key("binance", "ETH", "trade")]; exists {
		t.Fatal("snapshot contains unused key")
	}
	if _, exists := snap[k]; !exists {
		t.Fatal("snapshot missing the one key that issued")
	}
}

func TestSequencer_PeekDoesNotAdvance(t *testing.T) {
	s := sequencer.New()
	k := key("binance", "BTC", "trade")

	if _, ok := s.Peek(k); ok {
		t.Fatal("Peek on fresh key returned ok=true, want false")
	}
	s.Next(k) // 0
	if v, ok := s.Peek(k); !ok || v != 0 {
		t.Fatalf("Peek after one Next = (%d, %v), want (0, true)", v, ok)
	}
	if got := s.Next(k); got != 1 {
		t.Fatalf("Next after Peek = %d, want 1 (Peek must not advance)", got)
	}
}

func TestSequencer_ConcurrentSafe(t *testing.T) {
	s := sequencer.New()
	k := key("binance", "BTC/USDT-spot", "observation.trade")
	const goroutines = 50
	const perG = 100
	total := goroutines * perG

	var wg sync.WaitGroup
	values := make(chan int64, total)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				values <- s.Next(k)
			}
		}()
	}
	wg.Wait()
	close(values)

	seen := make(map[int64]struct{}, total)
	for v := range values {
		if _, dup := seen[v]; dup {
			t.Fatalf("duplicate sequence value emitted under concurrent load: %d", v)
		}
		seen[v] = struct{}{}
	}
	if len(seen) != total {
		t.Fatalf("got %d unique values, want %d", len(seen), total)
	}
	// Range invariant: values in [0, total-1] with no gaps.
	for i := int64(0); i < int64(total); i++ {
		if _, ok := seen[i]; !ok {
			t.Fatalf("missing sequence value %d under concurrent load", i)
		}
	}
}

func TestSequencer_ConcurrentDistinctKeys(t *testing.T) {
	s := sequencer.New()
	const keys = 8
	const perKey = 200

	var wg sync.WaitGroup
	wg.Add(keys)
	for kIdx := 0; kIdx < keys; kIdx++ {
		k := key("binance", "INSTR", "type-"+string(rune('A'+kIdx)))
		go func(k sequencer.StreamKey) {
			defer wg.Done()
			for i := 0; i < perKey; i++ {
				s.Next(k)
			}
		}(k)
	}
	wg.Wait()

	for kIdx := 0; kIdx < keys; kIdx++ {
		k := key("binance", "INSTR", "type-"+string(rune('A'+kIdx)))
		v, ok := s.Peek(k)
		if !ok {
			t.Fatalf("key %v missing from Peek", k)
		}
		if v != perKey-1 {
			t.Errorf("key %v last seq = %d, want %d", k, v, perKey-1)
		}
	}
}
