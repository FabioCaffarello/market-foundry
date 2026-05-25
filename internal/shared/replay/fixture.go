package replay

import (
	"time"

	envv1 "internal/shared/contracts/envelope/v1"
)

// fixtureRecord is the serialization form of a CanonicalEvent in
// a replay fixture. Fields are snake_case to match ADR-0017's
// proto envelope wire conventions; timestamps are epoch
// nanoseconds (not RFC3339 strings) so byte-stability is
// preserved across time-of-day and locale; ts_exchange is
// nullable to preserve ADR-0017's presence-significant semantics
// (nil = "venue did not provide", distinct from epoch 1970).
//
// This struct is intentionally unexported; CanonicalEvent
// remains the foundry-native domain projection per H-3.b's
// ADR-0017 Acceptance, and the fixture format is desacoplado de
// its evolution — adding a field to CanonicalEvent in a future
// onda updates toFixture/fromFixture here without disturbing the
// already-serialized fixture format.
type fixtureRecord struct {
	Type           string `json:"type"`
	Version        int32  `json:"version"`
	Venue          string `json:"venue"`
	Instrument     string `json:"instrument"`
	TsExchange     *int64 `json:"ts_exchange"`
	TsIngest       int64  `json:"ts_ingest"`
	Seq            int64  `json:"seq"`
	IdempotencyKey string `json:"idempotency_key"`
	Payload        []byte `json:"payload"`
}

// toFixture converts a CanonicalEvent into the fixture
// serialization form. Empty payload is normalized to []byte{}
// (matches proto3 semantics: nil and []byte{} are equivalent
// at the wire level). Timestamps are converted to epoch
// nanoseconds, discarding monotonic-clock reading and timezone
// (replay is absolute-time, timezone-independent).
func toFixture(ce envv1.CanonicalEvent) fixtureRecord {
	fr := fixtureRecord{
		Type:           ce.Type,
		Version:        ce.Version,
		Venue:          ce.Venue,
		Instrument:     ce.Instrument,
		TsIngest:       ce.TsIngest.UnixNano(),
		Seq:            ce.Seq,
		IdempotencyKey: ce.IdempotencyKey,
		Payload:        ce.Payload,
	}
	if ce.TsExchange != nil {
		ns := ce.TsExchange.UnixNano()
		fr.TsExchange = &ns
	}
	if len(fr.Payload) == 0 {
		fr.Payload = []byte{}
	}
	return fr
}

// fromFixture converts a fixture record back into a
// CanonicalEvent. Timestamps are reconstructed in UTC; payload
// preserves the (normalized) empty representation.
func fromFixture(fr fixtureRecord) envv1.CanonicalEvent {
	ce := envv1.CanonicalEvent{
		Type:           fr.Type,
		Version:        fr.Version,
		Venue:          fr.Venue,
		Instrument:     fr.Instrument,
		TsIngest:       time.Unix(0, fr.TsIngest).UTC(),
		Seq:            fr.Seq,
		IdempotencyKey: fr.IdempotencyKey,
		Payload:        fr.Payload,
	}
	if fr.TsExchange != nil {
		ts := time.Unix(0, *fr.TsExchange).UTC()
		ce.TsExchange = &ts
	}
	if ce.Payload == nil {
		ce.Payload = []byte{}
	}
	return ce
}
