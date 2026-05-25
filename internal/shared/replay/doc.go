// Package replay provides record/replay infrastructure for
// CanonicalEvent sequences, satisfying ADR-0019 acceptance
// criterion 1 (record/replay infrastructure) and providing the
// substrate for INV-D3 (byte-identical replay) and INV-D4 (N=50
// byte-stability across runs).
//
// # Usage
//
// Record a sequence:
//
//	r := replay.NewRecorder()
//	for _, ce := range events {
//	    r.Record(ce)
//	}
//	if _, err := r.WriteTo(file); err != nil { /* ... */ }
//
// Replay a sequence:
//
//	p, err := replay.NewPlayer(file)
//	if err != nil { /* ... */ }
//	for {
//	    ce, ok := p.Next()
//	    if !ok { break }
//	    process(ce)
//	}
//
// # Fixture format
//
// Fixtures are JSON-lines (.jsonl): one JSON object per line,
// terminated by a single newline. The on-disk schema is the
// fixture-private fixtureRecord struct, with snake_case field
// names (type, version, venue, instrument, ts_exchange,
// ts_ingest, seq, idempotency_key, payload) and epoch-nanosecond
// timestamps. The format is intentionally human-reviewable: a
// reviewer can cat a fixture and a diff highlights the semantic
// change, which is essential when an intentional logic change
// invalidates a golden test.
//
// # Why stdlib encoding/json, not protojson
//
// Replay fixtures use stdlib encoding/json on an internal
// fixtureRecord struct, NOT protojson on envelope.v1.Envelope.
// Rationale: protojson.Marshal output is not stable between
// versions of google.golang.org/protobuf (per upstream docs). A
// transitive dependency bump would silently break all goldens
// and INV-D4 byte-stability. The stdlib json encoder is
// byte-stable given a fixed struct definition, which the
// fixtureRecord provides.
//
// # Payload normalization
//
// CanonicalEvent.Payload is treated as opaque bytes by the
// replay layer. Empty payloads (nil OR []byte{}) are normalized
// to []byte{} on recording so that the fixture format has a
// single canonical encoding for empty ("" in JSON, not "null").
// This matches proto3 semantics (nil and []byte{} are equivalent
// at the wire level; see proto/envelope/v1/envelope.proto's
// payload field comment) and removes a spurious source of
// fixture diff noise. Future consumers that interpret Payload
// semantically must ensure their own determinism (e.g., avoid
// map[string]interface{} in intermediate marshaling, which
// randomizes key order).
//
// # Determinism boundary
//
// The package itself is determinism-neutral: the same input
// bytes always produce the same CanonicalEvent sequence, and
// the same CanonicalEvent sequence always produces the same
// fixture bytes. Consumers drive determinism end-to-end by
// combining replay with deterministic clock.Clock and
// random.Source ports (ADR-0019 INV-D1) and a deterministic
// Sequencer (ADR-0020). Replay alone guarantees input
// determinism; the rest is producer/consumer discipline.
//
// # Concurrency
//
// Recorder and Player are NOT safe for concurrent use. Callers
// that need to record or replay from multiple goroutines must
// serialize externally. In practice, the typical usage pattern
// is single-threaded per fixture (one goroutine driving record
// or replay), so internal locking would add cost without
// proportionate benefit.
package replay
