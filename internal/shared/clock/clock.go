// Package clock provides a port for time injection, satisfying
// ADR-0019 INV-D1 (domain purity).
//
// Production code in internal/domain/ MUST receive time via the
// Clock interface and never call time.Now() directly. Tests and
// replay infrastructure inject deterministic Clock implementations
// (FixedClock, or a custom replay-driven clock) so that domain
// behaviour is byte-stable across runs.
//
// The raccoon-cli check determinism analyzer (delivered in Onda
// H-4 commits 7/8) enforces this statically on internal/domain/
// production code; *_test.go files are exempt from the static
// check because the real enforcement for tests is the determinism
// gates INV-D3/INV-D4 (golden tests + N=50 byte-stability).
package clock

import "time"

// Clock is the canonical time port. Domain code holds a Clock
// value and calls Now() to obtain the current instant. The Clock
// interface is small on purpose — adding more methods grows the
// injection surface; helpers (Since, Until, etc.) belong outside
// this package as pure functions over Clock.Now().
type Clock interface {
	Now() time.Time
}

// SystemClock returns the operating system's wall clock. This is
// the default production implementation; cmd/ binaries instantiate
// SystemClock{} at boot and thread it down through actor and
// adapter configurations.
type SystemClock struct{}

// Now returns the current wall-clock instant.
func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock returns the same instant for every call. Used by
// tests and replay infrastructure to drive deterministic time.
// Replay drivers may wrap FixedClock or implement their own Clock
// that advances on a recorded schedule.
type FixedClock struct {
	Instant time.Time
}

// Now returns the fixed instant configured at construction.
func (c FixedClock) Now() time.Time { return c.Instant }
