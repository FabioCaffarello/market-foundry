package ingest_test

import (
	"testing"

	"internal/application/ingest"
)

func TestParseBindingTopic(t *testing.T) {
	tests := []struct {
		topic  string
		source string
		symbol string
	}{
		{"binancef.btcusdt", "binancef", "btcusdt"},
		{"BINANCEF.ETHUSDT", "binancef", "ethusdt"},
		{"bybit.solusdt", "bybit", "solusdt"},
	}

	for _, tc := range tests {
		t.Run(tc.topic, func(t *testing.T) {
			target, prob := ingest.ParseBindingTopic(tc.topic)
			if prob != nil {
				t.Fatalf("unexpected error: %v", prob)
			}
			if target.Source != tc.source {
				t.Fatalf("expected source %s, got %s", tc.source, target.Source)
			}
			if target.Symbol != tc.symbol {
				t.Fatalf("expected symbol %s, got %s", tc.symbol, target.Symbol)
			}
		})
	}
}

func TestParseBindingTopic_Invalid(t *testing.T) {
	tests := []string{
		"",
		"binancef",
		".btcusdt",
		"binancef.",
		"...",
	}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			_, prob := ingest.ParseBindingTopic(topic)
			if prob == nil {
				t.Fatalf("expected error for topic %q", topic)
			}
		})
	}
}

func TestBindingTarget_Key(t *testing.T) {
	target := ingest.BindingTarget{Source: "binancef", Symbol: "btcusdt"}
	if key := target.Key(); key != "binancef.btcusdt" {
		t.Fatalf("expected binancef.btcusdt, got %s", key)
	}
}
