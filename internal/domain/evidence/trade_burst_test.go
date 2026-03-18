package evidence_test

import (
	"testing"
	"time"

	"internal/domain/evidence"
)

func validTradeBurst() evidence.EvidenceTradeBurst {
	now := time.Now().UTC().Truncate(60 * time.Second)
	return evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 150,
		BuyVolume:  "500000.00",
		SellVolume: "300000.00",
		OpenTime:   now,
		CloseTime:  now.Add(60 * time.Second),
		Burst:      true,
		Final:      true,
	}
}

func TestEvidenceTradeBurst_Validate(t *testing.T) {
	b := validTradeBurst()
	if prob := b.Validate(); prob != nil {
		t.Fatalf("expected valid trade burst, got: %v", prob)
	}
}

func TestEvidenceTradeBurst_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst
	}{
		{"empty source", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.Source = ""; return b }},
		{"empty symbol", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.Symbol = ""; return b }},
		{"zero timeframe", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.Timeframe = 0; return b }},
		{"negative timeframe", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.Timeframe = -1; return b }},
		{"empty buy_volume", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.BuyVolume = ""; return b }},
		{"empty sell_volume", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.SellVolume = ""; return b }},
		{"zero open_time", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.OpenTime = time.Time{}; return b }},
		{"zero close_time", func(b evidence.EvidenceTradeBurst) evidence.EvidenceTradeBurst { b.CloseTime = time.Time{}; return b }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.mutate(validTradeBurst())
			if prob := b.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEvidenceTradeBurst_Validate_CloseTimeAfterOpenTime(t *testing.T) {
	b := validTradeBurst()
	b.CloseTime = b.OpenTime.Add(-time.Second)
	if prob := b.Validate(); prob == nil {
		t.Fatal("expected validation error: close_time before open_time")
	}
}

func TestEvidenceTradeBurst_Validate_CloseTimeEqualOpenTime(t *testing.T) {
	b := validTradeBurst()
	b.CloseTime = b.OpenTime
	if prob := b.Validate(); prob == nil {
		t.Fatal("expected validation error: close_time equal to open_time")
	}
}

func TestEvidenceTradeBurst_Validate_BurstFalseIsValid(t *testing.T) {
	b := validTradeBurst()
	b.Burst = false
	if prob := b.Validate(); prob != nil {
		t.Fatalf("burst=false should be valid: %v", prob)
	}
}

func TestEvidenceTradeBurst_Validate_FinalFalseIsValid(t *testing.T) {
	b := validTradeBurst()
	b.Final = false
	if prob := b.Validate(); prob != nil {
		t.Fatalf("final=false should be valid: %v", prob)
	}
}

func TestEvidenceTradeBurst_Validate_ZeroTradeCountIsValid(t *testing.T) {
	// TradeCount=0 is valid — represents an empty window.
	b := validTradeBurst()
	b.TradeCount = 0
	if prob := b.Validate(); prob != nil {
		t.Fatalf("trade_count=0 should be valid: %v", prob)
	}
}

func TestEvidenceTradeBurst_Validate_MultipleErrors(t *testing.T) {
	// Completely empty trade burst should accumulate all validation issues.
	b := evidence.EvidenceTradeBurst{}
	prob := b.Validate()
	if prob == nil {
		t.Fatal("expected validation error for empty trade burst")
	}
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Fatalf("expected InvalidArgument code, got %s", prob.Code)
	}
}
