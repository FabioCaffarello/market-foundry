package store

import (
	"log/slog"
	"sync/atomic"
)

// ProjectionStats tracks projection outcomes for observability.
// All fields are safe for concurrent access via atomic operations.
// Shared across all projection actors to eliminate per-family stats duplication.
type ProjectionStats struct {
	Received        atomic.Int64 // total events received
	Materialized    atomic.Int64 // events written to KV
	SkippedStale    atomic.Int64 // skipped: existing entry is newer
	SkippedDedup    atomic.Int64 // skipped: duplicate key
	SkippedNonFinal atomic.Int64 // skipped: non-final events dropped
	Rejected        atomic.Int64 // rejected by domain validation
	Errors          atomic.Int64 // write errors
}

// CheckInvariant verifies that received == sum of all outcomes.
// Logs an error if the invariant is violated.
func (s *ProjectionStats) CheckInvariant(logger *slog.Logger) {
	received := s.Received.Load()
	sum := s.Materialized.Load() +
		s.SkippedStale.Load() +
		s.SkippedDedup.Load() +
		s.SkippedNonFinal.Load() +
		s.Rejected.Load() +
		s.Errors.Load()
	if received != sum {
		logger.Error("stats invariant violated: received != sum of outcomes",
			"received", received,
			"sum", sum,
			"materialized", s.Materialized.Load(),
			"skipped_stale", s.SkippedStale.Load(),
			"skipped_dedup", s.SkippedDedup.Load(),
			"skipped_non_final", s.SkippedNonFinal.Load(),
			"rejected", s.Rejected.Load(),
			"errors", s.Errors.Load(),
		)
	}
}

// Log emits all stats as a structured log line.
func (s *ProjectionStats) Log(logger *slog.Logger, family, bucket string) {
	logger.Info(family+" projection stats",
		"bucket", bucket,
		"received", s.Received.Load(),
		"materialized", s.Materialized.Load(),
		"skipped_stale", s.SkippedStale.Load(),
		"skipped_dedup", s.SkippedDedup.Load(),
		"skipped_non_final", s.SkippedNonFinal.Load(),
		"rejected", s.Rejected.Load(),
		"errors", s.Errors.Load(),
	)
}
