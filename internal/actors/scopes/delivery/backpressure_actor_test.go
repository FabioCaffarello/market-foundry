package delivery

import (
	"testing"

	deliverydomain "internal/domain/delivery"
)

// TestOfferDropOldest pins the DropOldest policy: a full buffer evicts
// its oldest queued frame to admit the incoming one (favoring
// freshness), and the bound still holds. Deterministic — no engine, no
// goroutine (the write loop is the only other consumer and isn't running
// here).
func TestOfferDropOldest(t *testing.T) {
	a := &SessionActor{
		out: make(chan []byte, 2),
		cfg: sessionConfig{policy: deliverydomain.DropOldest},
	}

	a.offer([]byte("a"))
	a.offer([]byte("b"))
	// buffer (cap 2) is full; DropOldest evicts "a" and admits "c".
	a.offer([]byte("c"))

	if got := len(a.out); got != 2 {
		t.Fatalf("buffered = %d, want 2", got)
	}
	if a.dropped != 1 {
		t.Fatalf("dropped = %d, want 1", a.dropped)
	}
	first := string(<-a.out)
	second := string(<-a.out)
	if first != "b" || second != "c" {
		t.Fatalf("buffer = [%s %s], want [b c] (oldest 'a' evicted)", first, second)
	}
}
