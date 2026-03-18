package evidence_test

import (
	"testing"
	"time"

	"internal/domain/evidence"
)

func validCandle() evidence.EvidenceCandle {
	now := time.Now().UTC().Truncate(60 * time.Second)
	return evidence.EvidenceCandle{
		Source:     "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Open:      "84521.30",
		High:      "84589.90",
		Low:       "84510.00",
		Close:     "84575.40",
		Volume:    "12.345",
		TradeCount: 87,
		OpenTime:  now,
		CloseTime: now.Add(60 * time.Second),
		Final:     true,
	}
}

func TestEvidenceCandle_Validate(t *testing.T) {
	c := validCandle()
	if prob := c.Validate(); prob != nil {
		t.Fatalf("expected valid candle, got: %v", prob)
	}
}

func TestEvidenceCandle_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		mutate func(evidence.EvidenceCandle) evidence.EvidenceCandle
	}{
		{"empty source", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Source = ""; return c }},
		{"empty symbol", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Symbol = ""; return c }},
		{"zero timeframe", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Timeframe = 0; return c }},
		{"negative timeframe", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Timeframe = -1; return c }},
		{"empty open", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Open = ""; return c }},
		{"empty high", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.High = ""; return c }},
		{"empty low", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Low = ""; return c }},
		{"empty close", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Close = ""; return c }},
		{"empty volume", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Volume = ""; return c }},
		{"zero open_time", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.OpenTime = time.Time{}; return c }},
		{"zero close_time", func(c evidence.EvidenceCandle) evidence.EvidenceCandle { c.CloseTime = time.Time{}; return c }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.mutate(validCandle())
			if prob := c.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEvidenceCandle_Validate_CloseTimeAfterOpenTime(t *testing.T) {
	c := validCandle()
	c.CloseTime = c.OpenTime.Add(-time.Second)
	if prob := c.Validate(); prob == nil {
		t.Fatal("expected validation error: close_time before open_time")
	}
}
