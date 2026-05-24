package v1

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestEnvelopeRoundTrip validates ADR-0017 acceptance criterion 3:
// the proto envelope serialises and deserialises through proto.Marshal
// / proto.Unmarshal with structural equivalence preserved.
func TestEnvelopeRoundTrip(t *testing.T) {
	tsExchange := int64(1_700_000_000_000_000_000)
	original := &Envelope{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     &tsExchange,
		TsIngest:       1_700_000_000_500_000_000,
		Seq:            42,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:42",
		Payload:        []byte("dummy-payload-bytes"),
	}

	encoded, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	recovered := &Envelope{}
	if err := proto.Unmarshal(encoded, recovered); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if !proto.Equal(original, recovered) {
		t.Fatalf("round-trip mismatch:\noriginal:  %+v\nrecovered: %+v", original, recovered)
	}
}

// TestEnvelopeRoundTrip_TsExchangeAbsent validates that the optional
// ts_exchange field per ADR-0017 (proto3 `optional` keyword) preserves
// genuine absence across round-trip — i.e., a nil pointer does not
// silently become a zero value, which would collide with epoch
// 1970-01-01T00:00:00Z and break the zero-as-sentinel disambiguation
// the ADR's "Decision" table specifies.
func TestEnvelopeRoundTrip_TsExchangeAbsent(t *testing.T) {
	original := &Envelope{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     nil, // explicit absence
		TsIngest:       1_700_000_000_500_000_000,
		Seq:            7,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:7",
		Payload:        []byte("synthetic-event-no-venue-time"),
	}

	encoded, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	recovered := &Envelope{}
	if err := proto.Unmarshal(encoded, recovered); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if recovered.TsExchange != nil {
		t.Fatalf("expected TsExchange to remain nil after round-trip; got %d", *recovered.TsExchange)
	}
	if !proto.Equal(original, recovered) {
		t.Fatalf("round-trip mismatch on optional-absent envelope")
	}
}

// TestEnvelopeByteStability validates a contract aligned with ADR-0019
// INV-D4: marshalling the same envelope N times must produce the same
// bytes every time. Envelope carries no map fields, so byte-stability
// is achievable; a future regression (e.g., addition of a map field
// without a deterministic encoding strategy) is caught here.
func TestEnvelopeByteStability(t *testing.T) {
	const iterations = 50

	tsExchange := int64(1_700_000_000_000_000_000)
	env := &Envelope{
		Type:           "evidence.candle",
		Version:        1,
		Venue:          "binancef",
		Instrument:     "BTC/USDT-perpetual",
		TsExchange:     &tsExchange,
		TsIngest:       1_700_000_000_750_000_000,
		Seq:            100,
		IdempotencyKey: "binancef:BTC/USDT-perpetual:evidence.candle:100",
		Payload:        []byte("byte-stability-fixture"),
	}

	first, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("proto.Marshal (iter 0): %v", err)
	}

	for i := 1; i < iterations; i++ {
		encoded, err := proto.Marshal(env)
		if err != nil {
			t.Fatalf("proto.Marshal (iter %d): %v", i, err)
		}
		if !bytes.Equal(first, encoded) {
			t.Fatalf("byte-stability violated at iteration %d: encoding diverged from iter 0", i)
		}
	}
}
