package natssequencer

import (
	"context"
	"testing"

	"internal/shared/sequencer"
)

// store_unit_test.go — unit coverage for paths that don't require
// a running NATS server. Integration coverage for round-trip
// against a real NATS lives in store_roundtrip_test.go behind
// the integration build tag.

func TestLoadSnapshot_NilReceiver_ReturnsNotStarted(t *testing.T) {
	var s *Store
	_, err := s.LoadSnapshot(context.Background())
	if err == nil {
		t.Fatal("expected error on nil receiver")
	}
}

func TestLoadSnapshot_UnstartedStore_ReturnsNotStarted(t *testing.T) {
	s := NewStore("nats://localhost:4222", "ingest")
	_, err := s.LoadSnapshot(context.Background())
	if err == nil {
		t.Fatal("expected error on unstarted store")
	}
}

func TestSaveSnapshot_NilReceiver_ReturnsNotStarted(t *testing.T) {
	var s *Store
	err := s.SaveSnapshot(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on nil receiver")
	}
}

func TestSaveSnapshot_UnstartedStore_ReturnsNotStarted(t *testing.T) {
	s := NewStore("nats://localhost:4222", "ingest")
	err := s.SaveSnapshot(context.Background(), map[sequencer.StreamKey]int64{
		{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}: 0,
	})
	if err == nil {
		t.Fatal("expected error on unstarted store")
	}
}

func TestClose_NilReceiver_NoPanic(t *testing.T) {
	var s *Store
	if err := s.Close(); err != nil {
		t.Errorf("Close on nil receiver returned error: %v", err)
	}
}

func TestClose_UnstartedStore_NoPanic(t *testing.T) {
	s := NewStore("nats://localhost:4222", "ingest")
	if err := s.Close(); err != nil {
		t.Errorf("Close on unstarted store returned error: %v", err)
	}
}

func TestFormatKey_FollowsADR0020Format(t *testing.T) {
	sk := sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}
	got := formatKey("ingest", sk)
	want := "seq.ingest.binance.btcusdt.observation.trade"
	if got != want {
		t.Errorf("formatKey = %q, want %q", got, want)
	}
}

func TestFormatKey_EventTypeWithDots(t *testing.T) {
	sk := sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.book.snapshot"}
	got := formatKey("ingest", sk)
	want := "seq.ingest.binance.btcusdt.observation.book.snapshot"
	if got != want {
		t.Errorf("formatKey = %q, want %q", got, want)
	}
}

func TestParseKey_RoundTrip(t *testing.T) {
	cases := []struct {
		owner string
		key   sequencer.StreamKey
	}{
		{"ingest", sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}},
		{"derive", sequencer.StreamKey{Venue: "binancef", Instrument: "ethusdt-perp", EventType: "evidence.candle"}},
		{"execute", sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "execution.fill"}},
		{"ingest", sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.book.snapshot"}},
	}
	for _, tc := range cases {
		t.Run(tc.owner+"_"+tc.key.EventType, func(t *testing.T) {
			k := formatKey(tc.owner, tc.key)
			owner, sk, ok := parseKey(k)
			if !ok {
				t.Fatalf("parseKey(%q) returned ok=false", k)
			}
			if owner != tc.owner {
				t.Errorf("owner = %q, want %q", owner, tc.owner)
			}
			if sk != tc.key {
				t.Errorf("StreamKey = %+v, want %+v", sk, tc.key)
			}
		})
	}
}

func TestParseKey_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"foo",
		"seq.ingest.binance",                  // only 3 parts
		"seq.ingest.binance.btcusdt",          // only 4 parts
		"notseq.ingest.binance.btcusdt.trade", // wrong prefix
	}
	for _, k := range cases {
		t.Run(k, func(t *testing.T) {
			if _, _, ok := parseKey(k); ok {
				t.Errorf("parseKey(%q) returned ok=true, want false", k)
			}
		})
	}
}
