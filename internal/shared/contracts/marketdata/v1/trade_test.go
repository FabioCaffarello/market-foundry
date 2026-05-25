package v1

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestTradeV1RoundTrip validates the per-payload codegen pattern
// established by ADR-0018: a non-envelope schema (marketdata.trade
// pilot) serialises and deserialises through proto.Marshal /
// proto.Unmarshal with structural equivalence preserved.
func TestTradeV1RoundTrip(t *testing.T) {
	original := &TradeV1{
		Price:        29_312.45,
		Quantity:     0.0123,
		BuyerIsTaker: true,
		TradeId:      9_876_543_210,
	}

	encoded, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	recovered := &TradeV1{}
	if err := proto.Unmarshal(encoded, recovered); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if !proto.Equal(original, recovered) {
		t.Fatalf("round-trip mismatch:\noriginal:  %+v\nrecovered: %+v", original, recovered)
	}
}

// TestTradeV1ByteStability mirrors TestEnvelopeByteStability for the
// trade payload — ADR-0019 INV-D4 regression guard. TradeV1 carries
// no map fields; byte-stability is the steady-state contract.
func TestTradeV1ByteStability(t *testing.T) {
	const iterations = 50

	trade := &TradeV1{
		Price:        50_125.78,
		Quantity:     1.5,
		BuyerIsTaker: false,
		TradeId:      1_234_567_890,
	}

	first, err := proto.Marshal(trade)
	if err != nil {
		t.Fatalf("proto.Marshal (iter 0): %v", err)
	}

	for i := 1; i < iterations; i++ {
		encoded, err := proto.Marshal(trade)
		if err != nil {
			t.Fatalf("proto.Marshal (iter %d): %v", i, err)
		}
		if !bytes.Equal(first, encoded) {
			t.Fatalf("byte-stability violated at iteration %d", i)
		}
	}
}
