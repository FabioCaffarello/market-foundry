package evidence

import (
	"testing"
	"time"
)

func TestEvidenceVolume_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(60 * time.Second)
	valid := EvidenceVolume{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
		BuyVolume: "500000.00", SellVolume: "300000.00",
		TotalVolume: "800000.00", VWAP: "50000.12345678",
		TradeCount: 100,
		OpenTime: now, CloseTime: now.Add(60 * time.Second),
		Final: true,
	}

	if prob := valid.Validate(); prob != nil {
		t.Fatalf("expected valid, got: %s", prob.Message)
	}

	tests := []struct {
		name   string
		mutate func(*EvidenceVolume)
	}{
		{"empty source", func(v *EvidenceVolume) { v.Source = "" }},
		{"empty symbol", func(v *EvidenceVolume) { v.Symbol = "" }},
		{"zero timeframe", func(v *EvidenceVolume) { v.Timeframe = 0 }},
		{"negative timeframe", func(v *EvidenceVolume) { v.Timeframe = -1 }},
		{"empty buy_volume", func(v *EvidenceVolume) { v.BuyVolume = "" }},
		{"empty sell_volume", func(v *EvidenceVolume) { v.SellVolume = "" }},
		{"empty total_volume", func(v *EvidenceVolume) { v.TotalVolume = "" }},
		{"empty vwap", func(v *EvidenceVolume) { v.VWAP = "" }},
		{"zero open_time", func(v *EvidenceVolume) { v.OpenTime = time.Time{} }},
		{"zero close_time", func(v *EvidenceVolume) { v.CloseTime = time.Time{} }},
		{"close before open", func(v *EvidenceVolume) { v.CloseTime = v.OpenTime.Add(-time.Second) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vol := valid
			tt.mutate(&vol)
			if prob := vol.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
