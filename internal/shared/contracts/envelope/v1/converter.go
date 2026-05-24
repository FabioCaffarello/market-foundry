package v1

import (
	"fmt"
	"time"
)

// CanonicalEvent is the foundry-native domain projection of the
// canonical event envelope defined by ADR-0017. It mirrors the
// nine canonical fields of envelope.v1.Envelope using native Go
// types — time.Time vs int64 epoch nanoseconds, *time.Time vs
// proto optional, etc.
//
// CanonicalEvent is the boundary type between the proto wire
// format (internal/shared/contracts/envelope/v1/envelope.pb.go)
// and foundry domain code that produces or consumes mesh events.
// Direct use of *Envelope (proto) outside this package is
// discouraged; consumers should hold CanonicalEvent values and
// translate via FromProto / ToProto at the package boundary.
//
// This struct satisfies ADR-0017 acceptance criterion 4
// (post-2026-05-25 erratum): "internal/shared/contracts/envelope/
// v1/converter.go (or equivalent) translates between the proto
// envelope and the foundry's domain types".
type CanonicalEvent struct {
	Type           string
	Version        int32
	Venue          string
	Instrument     string
	TsExchange     *time.Time // optional per ADR-0017; nil = absent (distinct from epoch 1970)
	TsIngest       time.Time  // required; IsZero() reports invalid
	Seq            int64
	IdempotencyKey string
	Payload        []byte
}

// ToProto translates a CanonicalEvent into the proto wire
// representation. Returns an error if any required field is
// missing (Type, Venue, Instrument, TsIngest, IdempotencyKey,
// Version). Seq may legitimately be 0 (first event of a stream
// key). Payload may legitimately be empty (event types that
// explicitly define an empty payload per ADR-0017's payload
// comment in envelope/v1/envelope.proto). TsExchange is optional
// and absent when nil.
func ToProto(e CanonicalEvent) (*Envelope, error) {
	if e.Type == "" {
		return nil, fmt.Errorf("CanonicalEvent.Type is required")
	}
	if e.Version == 0 {
		return nil, fmt.Errorf("CanonicalEvent.Version is required (>= 1)")
	}
	if e.Venue == "" {
		return nil, fmt.Errorf("CanonicalEvent.Venue is required")
	}
	if e.Instrument == "" {
		return nil, fmt.Errorf("CanonicalEvent.Instrument is required")
	}
	if e.TsIngest.IsZero() {
		return nil, fmt.Errorf("CanonicalEvent.TsIngest is required")
	}
	if e.IdempotencyKey == "" {
		return nil, fmt.Errorf("CanonicalEvent.IdempotencyKey is required")
	}

	out := &Envelope{
		Type:           e.Type,
		Version:        e.Version,
		Venue:          e.Venue,
		Instrument:     e.Instrument,
		TsIngest:       e.TsIngest.UnixNano(),
		Seq:            e.Seq,
		IdempotencyKey: e.IdempotencyKey,
		Payload:        e.Payload,
	}
	if e.TsExchange != nil {
		ns := e.TsExchange.UnixNano()
		out.TsExchange = &ns
	}
	return out, nil
}

// FromProto translates a proto Envelope into the foundry-native
// CanonicalEvent. Returns an error if any required field is
// missing on the proto side — defence in depth: proto3 does not
// enforce required fields at the wire level, so a malformed
// envelope arriving from the mesh would otherwise propagate as
// silently-defaulted values into the domain layer.
//
// All timestamps are interpreted as UTC.
func FromProto(e *Envelope) (CanonicalEvent, error) {
	if e == nil {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope is nil")
	}
	if e.GetType() == "" {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.type is required")
	}
	if e.GetVersion() == 0 {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.version is required (>= 1)")
	}
	if e.GetVenue() == "" {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.venue is required")
	}
	if e.GetInstrument() == "" {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.instrument is required")
	}
	if e.GetTsIngest() == 0 {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.ts_ingest is required")
	}
	if e.GetIdempotencyKey() == "" {
		return CanonicalEvent{}, fmt.Errorf("envelope.v1.Envelope.idempotency_key is required")
	}

	out := CanonicalEvent{
		Type:           e.GetType(),
		Version:        e.GetVersion(),
		Venue:          e.GetVenue(),
		Instrument:     e.GetInstrument(),
		TsIngest:       time.Unix(0, e.GetTsIngest()).UTC(),
		Seq:            e.GetSeq(),
		IdempotencyKey: e.GetIdempotencyKey(),
		Payload:        e.GetPayload(),
	}
	if e.TsExchange != nil {
		ts := time.Unix(0, *e.TsExchange).UTC()
		out.TsExchange = &ts
	}
	return out, nil
}
