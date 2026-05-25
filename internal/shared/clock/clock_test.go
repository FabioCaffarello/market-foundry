package clock_test

import (
	"testing"
	"time"

	"internal/shared/clock"
)

func TestSystemClock_AdvancesMonotonically(t *testing.T) {
	c := clock.SystemClock{}
	t1 := c.Now()
	time.Sleep(time.Millisecond)
	t2 := c.Now()
	if !t2.After(t1) {
		t.Fatalf("expected t2 after t1; got t1=%s t2=%s", t1, t2)
	}
}

func TestFixedClock_ReturnsExactInstant(t *testing.T) {
	want := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	c := clock.FixedClock{Instant: want}
	got := c.Now()
	if !got.Equal(want) {
		t.Fatalf("Now() = %s, want %s", got, want)
	}
}

func TestFixedClock_StableAcrossCalls(t *testing.T) {
	want := time.Date(2026, 5, 25, 12, 0, 0, 42, time.UTC)
	c := clock.FixedClock{Instant: want}
	for i := 0; i < 100; i++ {
		got := c.Now()
		if !got.Equal(want) {
			t.Fatalf("call %d: Now() = %s, want %s", i, got, want)
		}
	}
}

func TestSystemClock_SatisfiesClockInterface(t *testing.T) {
	var _ clock.Clock = clock.SystemClock{}
}

func TestFixedClock_SatisfiesClockInterface(t *testing.T) {
	var _ clock.Clock = clock.FixedClock{}
}
