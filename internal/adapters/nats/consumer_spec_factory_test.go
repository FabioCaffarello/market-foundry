package nats

import (
	"testing"
	"time"
)

func TestNewConsumerSpec(t *testing.T) {
	spec := newConsumerSpec(
		"writer-signal-rsi",
		"signal.events.rsi.generated.>",
		"signal.events.v1.rsi_generated",
		"SIGNAL_EVENTS",
	)

	if spec.Durable != "writer-signal-rsi" {
		t.Errorf("Durable = %q, want %q", spec.Durable, "writer-signal-rsi")
	}
	if spec.Event.Subject != "signal.events.rsi.generated.>" {
		t.Errorf("Subject = %q, want %q", spec.Event.Subject, "signal.events.rsi.generated.>")
	}
	if spec.Event.Type != "signal.events.v1.rsi_generated" {
		t.Errorf("Type = %q, want %q", spec.Event.Type, "signal.events.v1.rsi_generated")
	}
	if spec.Event.Stream.Name != "SIGNAL_EVENTS" {
		t.Errorf("Stream.Name = %q, want %q", spec.Event.Stream.Name, "SIGNAL_EVENTS")
	}
	if spec.AckWait != 30*time.Second {
		t.Errorf("AckWait = %v, want %v", spec.AckWait, 30*time.Second)
	}
	if spec.MaxDeliver != 5 {
		t.Errorf("MaxDeliver = %d, want %d", spec.MaxDeliver, 5)
	}
}

func TestNewConsumerSpecDefaultsAreConsistent(t *testing.T) {
	spec := newConsumerSpec(
		"store-candle",
		"evidence.events.candle.sampled.>",
		"evidence.events.v1.candle_sampled",
		"EVIDENCE_EVENTS",
	)

	if spec.Durable != "store-candle" {
		t.Errorf("Durable = %q, want %q", spec.Durable, "store-candle")
	}
	if spec.Event.Subject != "evidence.events.candle.sampled.>" {
		t.Errorf("Subject = %q, want %q", spec.Event.Subject, "evidence.events.candle.sampled.>")
	}
	if spec.Event.Type != "evidence.events.v1.candle_sampled" {
		t.Errorf("Type = %q, want %q", spec.Event.Type, "evidence.events.v1.candle_sampled")
	}
	if spec.Event.Stream.Name != "EVIDENCE_EVENTS" {
		t.Errorf("Stream.Name = %q, want %q", spec.Event.Stream.Name, "EVIDENCE_EVENTS")
	}
	if spec.AckWait != 30*time.Second {
		t.Errorf("AckWait = %v, want 30s", spec.AckWait)
	}
	if spec.MaxDeliver != 5 {
		t.Errorf("MaxDeliver = %d, want 5", spec.MaxDeliver)
	}
}

func TestAllConsumerSpecFunctionsUseFactory(t *testing.T) {
	// Verify every public consumer spec function returns correct defaults.
	specs := []struct {
		name string
		fn   func() ConsumerSpec
	}{
		{"StoreCandleConsumer", StoreCandleConsumer},
		{"StoreTradeBurstConsumer", StoreTradeBurstConsumer},
		{"WriterCandleConsumer", WriterCandleConsumer},
		{"StoreVolumeConsumer", StoreVolumeConsumer},
		{"WriterRSISignalConsumer", WriterRSISignalConsumer},
		{"WriterEMASignalConsumer", WriterEMASignalConsumer},
		{"StoreRSISignalConsumer", StoreRSISignalConsumer},
		{"StoreEMACrossoverSignalConsumer", StoreEMACrossoverSignalConsumer},
		{"WriterRSIOversoldDecisionConsumer", WriterRSIOversoldDecisionConsumer},
		{"StoreRSIOversoldDecisionConsumer", StoreRSIOversoldDecisionConsumer},
		{"WriterMeanReversionEntryStrategyConsumer", WriterMeanReversionEntryStrategyConsumer},
		{"StoreMeanReversionEntryStrategyConsumer", StoreMeanReversionEntryStrategyConsumer},
		{"WriterPositionExposureRiskConsumer", WriterPositionExposureRiskConsumer},
		{"StorePositionExposureRiskConsumer", StorePositionExposureRiskConsumer},
		{"WriterPaperOrderExecutionConsumer", WriterPaperOrderExecutionConsumer},
		{"StorePaperOrderExecutionConsumer", StorePaperOrderExecutionConsumer},
		{"ExecuteVenueMarketOrderIntakeConsumer", ExecuteVenueMarketOrderIntakeConsumer},
		{"StoreVenueMarketOrderFillConsumer", StoreVenueMarketOrderFillConsumer},
	}

	for _, tc := range specs {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.fn()
			if spec.AckWait != 30*time.Second {
				t.Errorf("AckWait = %v, want 30s", spec.AckWait)
			}
			if spec.MaxDeliver != 5 {
				t.Errorf("MaxDeliver = %d, want 5", spec.MaxDeliver)
			}
			if spec.Durable == "" {
				t.Error("Durable is empty")
			}
			if spec.Event.Subject == "" {
				t.Error("Subject is empty")
			}
			if spec.Event.Type == "" {
				t.Error("Type is empty")
			}
			if spec.Event.Stream.Name == "" {
				t.Error("Stream.Name is empty")
			}
		})
	}
}
