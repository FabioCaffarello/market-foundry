// Package sequencer implements ADR-0020's per-stream-key
// monotonic sequence counter.
//
// The Sequencer produces a monotonic `seq` for each stream key
// (venue, instrument, event_type). Within a stream key,
// `seq(n+1) > seq(n)` holds always (INV-D2 of ADR-0019).
// Sequence spaces are independent across keys; no cross-key
// ordering is provided.
//
// Persistence is delegated to a separate adapter
// (internal/adapters/nats/natssequencer/) that snapshots and
// restores via the NATS KV bucket SEQUENCER_STATE_LATEST per
// ADR-0020. This package is in-memory only; durability is the
// caller's responsibility (boot: load via Restore; periodic
// flush: call Snapshot and write to KV).
//
// # Single-writer ownership
//
// Per ADR-0008, each NATS stream has exactly one writer binary.
// The Sequencer is owned by that writer for the keys it
// produces; double-allocation is prevented by construction. A
// single Sequencer instance per owner binary suffices.
//
// # Concurrency
//
// Sequencer is safe for concurrent use. Next, Snapshot, and
// Restore acquire an internal mutex.
package sequencer

import "sync"

// StreamKey is the (venue, instrument, event_type) tuple that
// addresses a per-key monotonic sequence space, per ADR-0020.
// Use value-equality semantics: two StreamKey values with equal
// fields compare as map-key-equivalent.
type StreamKey struct {
	Venue      string
	Instrument string
	EventType  string
}

// Sequencer issues monotonic int64 sequence numbers per
// StreamKey. The zero value is NOT usable; construct via New.
type Sequencer struct {
	mu      sync.Mutex
	highest map[StreamKey]int64 // last issued value per key (-1 = no events issued yet)
}

// New returns a Sequencer with no recorded keys. The first Next
// call for any key returns 0; subsequent calls for the same key
// return 1, 2, 3, ... unless Restore is called.
func New() *Sequencer {
	return &Sequencer{highest: make(map[StreamKey]int64)}
}

// Next returns the next sequence number for key and advances the
// internal high-water mark. Within a single key, the sequence
// returned by call n+1 is strictly greater than the sequence
// returned by call n (INV-D2). Sequence spaces are independent
// across keys.
func (s *Sequencer) Next(key StreamKey) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.highest[key]
	var next int64
	if ok {
		next = last + 1
	} else {
		next = 0
	}
	s.highest[key] = next
	return next
}

// Snapshot returns a copy of the current high-water marks. The
// returned map is a fresh allocation; callers may modify it
// freely without affecting the Sequencer. Keys that have never
// issued a sequence are absent from the snapshot.
//
// The snapshot is the durability primitive: persistence
// adapters serialize this map to KV; on restart, the recovered
// map is passed to Restore.
func (s *Sequencer) Snapshot() map[StreamKey]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[StreamKey]int64, len(s.highest))
	for k, v := range s.highest {
		out[k] = v
	}
	return out
}

// Restore replaces the internal high-water marks with the given
// snapshot. Subsequent Next calls return snapshot[key] + 1 for
// each restored key. Keys present in the Sequencer but absent
// from the snapshot are dropped — Restore is a replace, not a
// merge.
//
// Restore is intended for boot-time recovery: load the KV
// snapshot, hand it to Restore, then resume issuing. ADR-0020
// recovery semantics permit a small redundant-emit window if
// the last few values were not flushed to KV before the prior
// crash; downstream dedup via idempotency_key absorbs it.
func (s *Sequencer) Restore(snapshot map[StreamKey]int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.highest = make(map[StreamKey]int64, len(snapshot))
	for k, v := range snapshot {
		s.highest[k] = v
	}
}

// Peek returns the last issued sequence number for key and a
// boolean indicating whether the key has issued any sequences
// yet. Does not advance the internal state. Useful for
// telemetry and tests; not part of ADR-0020's normative
// contract.
func (s *Sequencer) Peek(key StreamKey) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.highest[key]
	return v, ok
}
