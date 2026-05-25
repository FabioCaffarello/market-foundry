// Package random provides a port for randomness injection,
// satisfying ADR-0019 INV-D1 (domain purity).
//
// Production code in internal/domain/ MUST receive randomness via
// the Source interface and never call math/rand or crypto/rand
// directly. Tests and replay infrastructure inject SeededSource
// instances so that domain behaviour is byte-stable across runs.
//
// The raccoon-cli check determinism analyzer (delivered in Onda
// H-4 commits 7/8) enforces this statically on internal/domain/
// production code; *_test.go files are exempt for the same reason
// documented in internal/shared/clock.
package random

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mathrand "math/rand/v2"
)

// Source is the canonical randomness port. The method set is small
// on purpose — extra helpers (Intn, Perm, Shuffle) compose from
// Int63/Float64 in caller code.
type Source interface {
	// Int63 returns a non-negative pseudo-random 63-bit integer
	// as an int64. The high bit is always zero.
	Int63() int64
	// Float64 returns a pseudo-random float64 in [0.0, 1.0).
	Float64() float64
	// Bytes returns n pseudo-random bytes. The returned slice is
	// freshly allocated; callers may modify it freely.
	Bytes(n int) []byte
}

// SystemSource is a Source seeded from crypto/rand at construction.
// Each instance carries its own internal generator; concurrent use
// of a single instance from multiple goroutines is NOT safe — the
// caller is responsible for serialization if needed (callers that
// hold per-actor instances do not need to coordinate).
type SystemSource struct {
	r *mathrand.Rand
}

// NewSystemSource returns a SystemSource seeded from crypto/rand.
// Panics if crypto/rand is unavailable, which on supported
// platforms (linux/darwin/windows) indicates a system fault that
// would prevent the rest of the binary from operating safely.
func NewSystemSource() *SystemSource {
	var seed [16]byte
	if _, err := cryptorand.Read(seed[:]); err != nil {
		panic("random: crypto/rand.Read failed: " + err.Error())
	}
	s1 := binary.LittleEndian.Uint64(seed[0:8])
	s2 := binary.LittleEndian.Uint64(seed[8:16])
	return &SystemSource{r: mathrand.New(mathrand.NewPCG(s1, s2))}
}

// Int63 returns a non-negative pseudo-random 63-bit integer.
func (s *SystemSource) Int63() int64 { return int63From(s.r) }

// Float64 returns a pseudo-random float64 in [0.0, 1.0).
func (s *SystemSource) Float64() float64 { return s.r.Float64() }

// Bytes returns n pseudo-random bytes.
func (s *SystemSource) Bytes(n int) []byte { return bytesFrom(s.r, n) }

// SeededSource is a Source seeded from a deterministic uint64.
// Same seed always produces the same byte-for-byte sequence.
// Intended for tests, fixtures, and replay drivers.
type SeededSource struct {
	r *mathrand.Rand
}

// NewSeededSource returns a SeededSource initialized with the
// given seed. The companion seed for the PCG generator is derived
// deterministically so a single uint64 seed parameter fully
// determines the sequence.
func NewSeededSource(seed uint64) *SeededSource {
	return &SeededSource{r: mathrand.New(mathrand.NewPCG(seed, seed^0xa5a5a5a5a5a5a5a5))}
}

// Int63 returns a non-negative pseudo-random 63-bit integer.
func (s *SeededSource) Int63() int64 { return int63From(s.r) }

// Float64 returns a pseudo-random float64 in [0.0, 1.0).
func (s *SeededSource) Float64() float64 { return s.r.Float64() }

// Bytes returns n pseudo-random bytes.
func (s *SeededSource) Bytes(n int) []byte { return bytesFrom(s.r, n) }

func int63From(r *mathrand.Rand) int64 {
	return int64(r.Uint64() & ((1 << 63) - 1))
}

func bytesFrom(r *mathrand.Rand, n int) []byte {
	if n <= 0 {
		return []byte{}
	}
	out := make([]byte, n)
	for i := 0; i < n; i += 8 {
		u := r.Uint64()
		for j := 0; j < 8 && i+j < n; j++ {
			out[i+j] = byte(u >> uint(j*8))
		}
	}
	return out
}
