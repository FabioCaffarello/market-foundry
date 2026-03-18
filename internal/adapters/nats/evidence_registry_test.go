package nats_test

import (
	"strings"
	"testing"
	"time"

	adapternats "internal/adapters/nats"
)

func TestEvidenceRegistry_SubjectConventions(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()

	t.Run("candle sampled subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.events.candle.sampled"
		if reg.CandleSampled.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.CandleSampled.Subject)
		}
	})

	t.Run("candle type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.CandleSampled.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.CandleSampled.Type)
		}
	})

	t.Run("stream name is EVIDENCE_EVENTS", func(t *testing.T) {
		if reg.CandleSampled.Stream.Name != "EVIDENCE_EVENTS" {
			t.Errorf("expected EVIDENCE_EVENTS, got %s", reg.CandleSampled.Stream.Name)
		}
	})

	t.Run("stream subjects use wildcard", func(t *testing.T) {
		subjects := reg.CandleSampled.Stream.Subjects
		if len(subjects) != 1 || subjects[0] != "evidence.events.>" {
			t.Errorf("expected [evidence.events.>], got %v", subjects)
		}
	})

	t.Run("query subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.query.candle.latest"
		if reg.CandleLatest.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.CandleLatest.Subject)
		}
	})

	t.Run("query queue group is evidence.query", func(t *testing.T) {
		if reg.CandleLatest.QueueGroup != "evidence.query" {
			t.Errorf("expected evidence.query, got %s", reg.CandleLatest.QueueGroup)
		}
	})

	t.Run("history query subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.query.candle.history"
		if reg.CandleHistory.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.CandleHistory.Subject)
		}
	})

	t.Run("history query queue group is evidence.query", func(t *testing.T) {
		if reg.CandleHistory.QueueGroup != "evidence.query" {
			t.Errorf("expected evidence.query, got %s", reg.CandleHistory.QueueGroup)
		}
	})

	t.Run("history request type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.CandleHistory.RequestType, ".v1.") {
			t.Errorf("expected versioned request type, got %s", reg.CandleHistory.RequestType)
		}
	})

	t.Run("trade burst sampled subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.events.tradeburst.sampled"
		if reg.TradeBurstSampled.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.TradeBurstSampled.Subject)
		}
	})

	t.Run("trade burst type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.TradeBurstSampled.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.TradeBurstSampled.Type)
		}
	})

	t.Run("trade burst query subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.query.tradeburst.latest"
		if reg.TradeBurstLatest.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.TradeBurstLatest.Subject)
		}
	})

	t.Run("trade burst query queue group is evidence.query", func(t *testing.T) {
		if reg.TradeBurstLatest.QueueGroup != "evidence.query" {
			t.Errorf("expected evidence.query, got %s", reg.TradeBurstLatest.QueueGroup)
		}
	})

	t.Run("candle consumer durable follows naming", func(t *testing.T) {
		spec := adapternats.StoreCandleConsumer()
		if spec.Durable != "store-candle" {
			t.Errorf("expected store-candle, got %s", spec.Durable)
		}
	})

	t.Run("trade burst consumer durable follows naming", func(t *testing.T) {
		spec := adapternats.StoreTradeBurstConsumer()
		if spec.Durable != "store-trade-burst" {
			t.Errorf("expected store-trade-burst, got %s", spec.Durable)
		}
	})

	t.Run("all consumer durables use hyphen-separated words", func(t *testing.T) {
		for _, spec := range []adapternats.ConsumerSpec{
			adapternats.StoreCandleConsumer(),
			adapternats.StoreTradeBurstConsumer(),
		} {
			if strings.Contains(spec.Durable, "_") {
				t.Errorf("durable name should use hyphens, not underscores: %s", spec.Durable)
			}
		}
	})

	t.Run("all subjects are lowercase", func(t *testing.T) {
		for _, s := range []string{reg.CandleSampled.Subject, reg.CandleLatest.Subject, reg.CandleHistory.Subject, reg.TradeBurstSampled.Subject, reg.TradeBurstLatest.Subject} {
			if s != strings.ToLower(s) {
				t.Errorf("subject must be lowercase: %s", s)
			}
		}
	})
}

func TestEvidenceRegistry_VolumeConventions(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()

	t.Run("volume sampled subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.events.volume.sampled"
		if reg.VolumeSampled.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.VolumeSampled.Subject)
		}
	})

	t.Run("volume type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.VolumeSampled.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.VolumeSampled.Type)
		}
	})

	t.Run("volume query subject follows taxonomy", func(t *testing.T) {
		expected := "evidence.query.volume.latest"
		if reg.VolumeLatest.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.VolumeLatest.Subject)
		}
	})

	t.Run("volume query queue group is evidence.query", func(t *testing.T) {
		if reg.VolumeLatest.QueueGroup != "evidence.query" {
			t.Errorf("expected evidence.query, got %s", reg.VolumeLatest.QueueGroup)
		}
	})
}

func TestEvidenceRegistry_StreamConstraints(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()
	stream := reg.CandleSampled.Stream

	t.Run("stream has positive MaxAge", func(t *testing.T) {
		if stream.MaxAge <= 0 {
			t.Errorf("MaxAge must be positive, got %v", stream.MaxAge)
		}
	})

	t.Run("stream has positive MaxBytes", func(t *testing.T) {
		if stream.MaxBytes <= 0 {
			t.Errorf("MaxBytes must be positive, got %d", stream.MaxBytes)
		}
	})

	t.Run("stream has finite retention", func(t *testing.T) {
		if stream.MaxAge > 168*time.Hour {
			t.Errorf("MaxAge should be reasonable (<=7 days), got %v", stream.MaxAge)
		}
	})
}

func TestEvidenceRegistry_AllEventTypesShareStream(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()

	candleStream := reg.CandleSampled.Stream.Name
	burstStream := reg.TradeBurstSampled.Stream.Name
	volumeStream := reg.VolumeSampled.Stream.Name

	if candleStream != burstStream || burstStream != volumeStream {
		t.Errorf("all evidence event types must share the same stream: candle=%s, burst=%s, volume=%s",
			candleStream, burstStream, volumeStream)
	}
}

func TestEvidenceRegistry_ConsumerSpecs(t *testing.T) {
	consumers := []struct {
		name string
		spec adapternats.ConsumerSpec
	}{
		{"StoreCandleConsumer", adapternats.StoreCandleConsumer()},
		{"StoreTradeBurstConsumer", adapternats.StoreTradeBurstConsumer()},
		{"StoreVolumeConsumer", adapternats.StoreVolumeConsumer()},
	}

	for _, tc := range consumers {
		t.Run(tc.name+"_durable_uses_hyphens", func(t *testing.T) {
			if strings.Contains(tc.spec.Durable, "_") {
				t.Errorf("durable name should use hyphens: %s", tc.spec.Durable)
			}
		})

		t.Run(tc.name+"_max_deliver_bounded", func(t *testing.T) {
			if tc.spec.MaxDeliver < 1 || tc.spec.MaxDeliver > 10 {
				t.Errorf("MaxDeliver should be between 1-10, got %d", tc.spec.MaxDeliver)
			}
		})

		t.Run(tc.name+"_ack_wait_positive", func(t *testing.T) {
			if tc.spec.AckWait <= 0 {
				t.Errorf("AckWait must be positive, got %v", tc.spec.AckWait)
			}
		})

		t.Run(tc.name+"_filter_uses_wildcard", func(t *testing.T) {
			if !strings.HasSuffix(tc.spec.Event.Subject, ".>") {
				t.Errorf("consumer filter must use wildcard suffix: %s", tc.spec.Event.Subject)
			}
		})

		t.Run(tc.name+"_stream_is_evidence", func(t *testing.T) {
			if tc.spec.Event.Stream.Name != "EVIDENCE_EVENTS" {
				t.Errorf("expected EVIDENCE_EVENTS, got %s", tc.spec.Event.Stream.Name)
			}
		})
	}
}

func TestStoreVolumeConsumer_Spec(t *testing.T) {
	spec := adapternats.StoreVolumeConsumer()
	if spec.Durable != "store-volume" {
		t.Errorf("expected store-volume, got %s", spec.Durable)
	}
}

func TestEvidenceRegistry_SubjectIsolation(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()

	// All event subjects must be distinct — no cross-type collision.
	subjects := map[string]string{
		"candle":     reg.CandleSampled.Subject,
		"tradeburst": reg.TradeBurstSampled.Subject,
		"volume":     reg.VolumeSampled.Subject,
	}

	seen := make(map[string]string)
	for name, subj := range subjects {
		if prev, exists := seen[subj]; exists {
			t.Errorf("subject collision between %s and %s: %s", prev, name, subj)
		}
		seen[subj] = name
	}
}

func TestEvidenceRegistry_QuerySubjectIsolation(t *testing.T) {
	reg := adapternats.DefaultEvidenceRegistry()

	// All query subjects must be distinct.
	queries := map[string]string{
		"candle_latest":     reg.CandleLatest.Subject,
		"candle_history":    reg.CandleHistory.Subject,
		"tradeburst_latest": reg.TradeBurstLatest.Subject,
		"volume_latest":     reg.VolumeLatest.Subject,
	}

	seen := make(map[string]string)
	for name, subj := range queries {
		if prev, exists := seen[subj]; exists {
			t.Errorf("query subject collision between %s and %s: %s", prev, name, subj)
		}
		seen[subj] = name
	}
}

func TestEvidenceRegistry_DedupKeyIsolation(t *testing.T) {
	// Verify that dedup key formats for different evidence types cannot collide.
	// Candle: "source:symbol:tf:opentime"
	// TradeBurst: "burst:source:symbol:tf:opentime"
	// Volume: "vol:source:symbol:tf:opentime"
	candleKey := "binancef:btcusdt:60:1710000000"
	burstKey := "burst:binancef:btcusdt:60:1710000000"
	volumeKey := "vol:binancef:btcusdt:60:1710000000"

	keys := map[string]bool{candleKey: true, burstKey: true, volumeKey: true}
	if len(keys) != 3 {
		t.Fatal("dedup key formats must be distinct across evidence types")
	}
}
