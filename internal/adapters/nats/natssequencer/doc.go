// Package natssequencer is the NATS JetStream KV adapter for the
// Sequencer's snapshot/restore primitives.
//
// # Bucket
//
// `SEQUENCER_STATE_LATEST` (declared in this package, created by
// Store.Start). One entry per StreamKey; the value is the
// highest issued seq for that key (decimal-string-encoded
// int64).
//
// Per ADR-0008 single-writer invariant, the bucket is owned by
// the writer binary that issues seq for the keys it produces.
// Concretely:
//
//   - `ingest` owns seq for OBSERVATION_EVENTS keys.
//   - `derive` owns seq for evidence / signal / decision /
//     strategy / risk / execution streams.
//   - `execute` owns seq for fill / rejection / session streams.
//   - `configctl` owns seq for CONFIGCTL_EVENTS.
//
// In H-4, only the bucket declaration and snapshot/restore
// primitives are introduced; wiring per writer binary lands in
// later ondas after the Sequencer is integrated into each
// writer's hot path.
//
// # Key format
//
// Per ADR-0020:
//
//	seq.{owner_binary}.{venue}.{instrument}.{event_type}
//
// Components MUST NOT contain dots EXCEPT event_type, which may
// (the foundry's event types are dotted: `observation.trade`,
// `observation.book.snapshot`, etc.). The parser in this package
// applies the convention "first four dot-separated tokens are
// seq/owner/venue/instrument; the rest rejoin as event_type" —
// see parseKey for the boundary.
//
// # Cadence
//
// This adapter exposes primitives (LoadSnapshot, SaveSnapshot);
// the cadence policy ("flush every N events OR T duration,
// whichever first" per ADR-0020 non-goals) is the writer
// binary's responsibility, not the Store's. Keeping the
// primitives policy-free lets each owner choose a cadence
// appropriate to its event rate (ingest is high-rate, configctl
// is rare).
//
// # Concurrency
//
// Store is safe for use from multiple goroutines after Start
// returns. Underlying jetstream.KeyValue handles concurrent
// Put/Get safely; Store itself adds no internal state requiring
// further synchronization.
package natssequencer
