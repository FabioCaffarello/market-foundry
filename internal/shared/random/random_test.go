package random_test

import (
	"bytes"
	"testing"

	"internal/shared/random"
)

func TestSystemSource_SatisfiesSourceInterface(t *testing.T) {
	var _ random.Source = random.NewSystemSource()
}

func TestSeededSource_SatisfiesSourceInterface(t *testing.T) {
	var _ random.Source = random.NewSeededSource(0)
}

func TestSeededSource_DeterministicAcrossInstances(t *testing.T) {
	const seed uint64 = 0x12345
	a := random.NewSeededSource(seed)
	b := random.NewSeededSource(seed)

	for i := 0; i < 1000; i++ {
		if a.Int63() != b.Int63() {
			t.Fatalf("Int63 diverged at iteration %d", i)
		}
	}
	for i := 0; i < 1000; i++ {
		if a.Float64() != b.Float64() {
			t.Fatalf("Float64 diverged at iteration %d", i)
		}
	}
	for i := 0; i < 100; i++ {
		ba := a.Bytes(16)
		bb := b.Bytes(16)
		if !bytes.Equal(ba, bb) {
			t.Fatalf("Bytes(16) diverged at iteration %d: %x vs %x", i, ba, bb)
		}
	}
}

func TestSeededSource_DifferentSeedsProduceDifferentSequences(t *testing.T) {
	a := random.NewSeededSource(1)
	b := random.NewSeededSource(2)
	matches := 0
	for i := 0; i < 8; i++ {
		if a.Int63() == b.Int63() {
			matches++
		}
	}
	if matches > 2 {
		t.Fatalf("expected divergence between distinct seeds; got %d/8 matches", matches)
	}
}

func TestSystemSource_DistinctAcrossCalls(t *testing.T) {
	s := random.NewSystemSource()
	seen := make(map[int64]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		v := s.Int63()
		if v < 0 {
			t.Fatalf("Int63 returned negative value: %d", v)
		}
		seen[v] = struct{}{}
	}
	if len(seen) < 990 {
		t.Fatalf("expected near-unique values across 1000 calls; got %d unique", len(seen))
	}
}

func TestSystemSource_DistinctInstancesProduceDistinctSequences(t *testing.T) {
	a := random.NewSystemSource()
	b := random.NewSystemSource()
	matches := 0
	for i := 0; i < 8; i++ {
		if a.Int63() == b.Int63() {
			matches++
		}
	}
	if matches > 2 {
		t.Fatalf("expected distinct seeds to diverge; got %d/8 matches", matches)
	}
}

func TestSource_Float64_InUnitInterval(t *testing.T) {
	s := random.NewSeededSource(42)
	for i := 0; i < 1000; i++ {
		v := s.Float64()
		if v < 0.0 || v >= 1.0 {
			t.Fatalf("Float64 outside [0,1) at iteration %d: %v", i, v)
		}
	}
}

func TestSource_Bytes_LengthAndZeroCases(t *testing.T) {
	s := random.NewSeededSource(7)
	if got := s.Bytes(0); len(got) != 0 {
		t.Fatalf("Bytes(0) = %v, want empty", got)
	}
	if got := s.Bytes(-1); len(got) != 0 {
		t.Fatalf("Bytes(-1) = %v, want empty", got)
	}
	for _, n := range []int{1, 7, 8, 9, 16, 31, 32, 100} {
		got := s.Bytes(n)
		if len(got) != n {
			t.Fatalf("Bytes(%d) length = %d", n, len(got))
		}
	}
}

func TestSource_Int63_HighBitAlwaysZero(t *testing.T) {
	s := random.NewSeededSource(0xdeadbeef)
	for i := 0; i < 1000; i++ {
		v := s.Int63()
		if v < 0 {
			t.Fatalf("Int63 returned a value with high bit set at iteration %d: %x", i, uint64(v))
		}
	}
}
