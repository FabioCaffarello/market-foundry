package observation_test

import (
	"strings"
	"testing"
	"time"

	"internal/domain/observation"
)

func TestObservationTrade_Validate(t *testing.T) {
	valid := observation.ObservationTrade{
		Source:     "binancef",
		Symbol:    "btcusdt",
		Price:     "84521.30",
		Quantity:  "0.150",
		TradeID:   "4839201",
		BuyerMaker: false,
		Timestamp: time.Now().UTC(),
	}

	if prob := valid.Validate(); prob != nil {
		t.Fatalf("expected valid trade, got: %v", prob)
	}

	tests := []struct {
		name  string
		trade observation.ObservationTrade
		field string
	}{
		{"empty source", func() observation.ObservationTrade { v := valid; v.Source = ""; return v }(), "source"},
		{"empty symbol", func() observation.ObservationTrade { v := valid; v.Symbol = ""; return v }(), "symbol"},
		{"empty price", func() observation.ObservationTrade { v := valid; v.Price = ""; return v }(), "price"},
		{"empty quantity", func() observation.ObservationTrade { v := valid; v.Quantity = ""; return v }(), "quantity"},
		{"empty trade_id", func() observation.ObservationTrade { v := valid; v.TradeID = ""; return v }(), "trade_id"},
		{"zero timestamp", func() observation.ObservationTrade { v := valid; v.Timestamp = time.Time{}; return v }(), "timestamp"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prob := tc.trade.Validate()
			if prob == nil {
				t.Fatalf("expected validation error for field %s", tc.field)
			}
		})
	}
}

func TestObservationTrade_DeduplicationKey(t *testing.T) {
	trade := observation.ObservationTrade{Source: "binancef", TradeID: "123456"}
	key := trade.DeduplicationKey()
	if key != "binancef:123456" {
		t.Fatalf("expected binancef:123456, got: %s", key)
	}
}

func TestObservationTrade_DeduplicationKey_Uniqueness(t *testing.T) {
	// Different sources with same trade ID must produce different keys.
	t1 := observation.ObservationTrade{Source: "binancef", TradeID: "100"}
	t2 := observation.ObservationTrade{Source: "coinbase", TradeID: "100"}
	if t1.DeduplicationKey() == t2.DeduplicationKey() {
		t.Fatal("different sources with same trade_id must produce different dedup keys")
	}
}

func TestObservationTrade_DeduplicationKey_Format(t *testing.T) {
	trade := observation.ObservationTrade{Source: "binancef", TradeID: "99887766"}
	key := trade.DeduplicationKey()
	if !strings.Contains(key, ":") {
		t.Fatal("dedup key must use colon separator")
	}
	parts := strings.SplitN(key, ":", 2)
	if parts[0] != "binancef" || parts[1] != "99887766" {
		t.Fatalf("dedup key parts mismatch: %v", parts)
	}
}

func TestObservationTrade_Validate_MultipleErrors(t *testing.T) {
	// Completely empty trade should accumulate all validation issues.
	trade := observation.ObservationTrade{}
	prob := trade.Validate()
	if prob == nil {
		t.Fatal("expected validation error for empty trade")
	}
	// Should contain multiple issues (at least source, symbol, price, quantity, trade_id, timestamp).
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Fatalf("expected InvalidArgument code, got %s", prob.Code)
	}
}

func TestObservationTrade_Validate_BuyerMakerBooleans(t *testing.T) {
	// BuyerMaker is a bool — both values should produce valid trades.
	base := observation.ObservationTrade{
		Source: "binancef", Symbol: "btcusdt", Price: "100.0",
		Quantity: "1.0", TradeID: "1", Timestamp: time.Now().UTC(),
	}

	base.BuyerMaker = true
	if prob := base.Validate(); prob != nil {
		t.Fatalf("buyer_maker=true should be valid: %v", prob)
	}

	base.BuyerMaker = false
	if prob := base.Validate(); prob != nil {
		t.Fatalf("buyer_maker=false should be valid: %v", prob)
	}
}
