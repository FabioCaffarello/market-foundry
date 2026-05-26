package evidence_test

import (
	"testing"
	"time"

	"internal/domain/evidence"
	"internal/domain/instrument"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("test setup: failed to build BTC/USDT-perpetual: %v", prob)
	}
	return inst
}

func validCandle(t *testing.T) evidence.EvidenceCandle {
	t.Helper()
	now := time.Now().UTC().Truncate(60 * time.Second)
	return evidence.EvidenceCandle{
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Open:       "84521.30",
		High:       "84589.90",
		Low:        "84510.00",
		Close:      "84575.40",
		Volume:     "12.345",
		TradeCount: 87,
		OpenTime:   now,
		CloseTime:  now.Add(60 * time.Second),
		Final:      true,
	}
}

func TestEvidenceCandle_Validate(t *testing.T) {
	c := validCandle(t)
	if prob := c.Validate(); prob != nil {
		t.Fatalf("expected valid candle, got: %v", prob)
	}
}

func TestEvidenceCandle_VenueSymbol(t *testing.T) {
	c := validCandle(t)
	if got := c.VenueSymbol(); got != "btcusdt" {
		t.Fatalf("expected venue symbol btcusdt, got %s", got)
	}
}

func TestEvidenceCandle_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, evidence.EvidenceCandle) evidence.EvidenceCandle
	}{
		{"empty source", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Source = ""; return c }},
		{"zero instrument", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle {
			c.Instrument = instrument.CanonicalInstrument{}
			return c
		}},
		{"zero timeframe", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Timeframe = 0; return c }},
		{"negative timeframe", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Timeframe = -1; return c }},
		{"empty open", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Open = ""; return c }},
		{"empty high", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.High = ""; return c }},
		{"empty low", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Low = ""; return c }},
		{"empty close", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Close = ""; return c }},
		{"empty volume", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle { c.Volume = ""; return c }},
		{"zero open_time", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle {
			c.OpenTime = time.Time{}
			return c
		}},
		{"zero close_time", func(t *testing.T, c evidence.EvidenceCandle) evidence.EvidenceCandle {
			c.CloseTime = time.Time{}
			return c
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.mutate(t, validCandle(t))
			if prob := c.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEvidenceCandle_Validate_CloseTimeAfterOpenTime(t *testing.T) {
	c := validCandle(t)
	c.CloseTime = c.OpenTime.Add(-time.Second)
	if prob := c.Validate(); prob == nil {
		t.Fatal("expected validation error: close_time before open_time")
	}
}
