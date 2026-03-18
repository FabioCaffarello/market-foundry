package nats_test

import (
	"strings"
	"testing"

	adapternats "internal/adapters/nats"
)

func TestSignalRegistry_SubjectConventions(t *testing.T) {
	reg := adapternats.DefaultSignalRegistry()

	t.Run("rsi generated subject follows taxonomy", func(t *testing.T) {
		expected := "signal.events.rsi.generated"
		if reg.RSIGenerated.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.RSIGenerated.Subject)
		}
	})

	t.Run("rsi type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSIGenerated.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.RSIGenerated.Type)
		}
	})

	t.Run("stream name is SIGNAL_EVENTS", func(t *testing.T) {
		if reg.RSIGenerated.Stream.Name != "SIGNAL_EVENTS" {
			t.Errorf("expected SIGNAL_EVENTS, got %s", reg.RSIGenerated.Stream.Name)
		}
	})

	t.Run("stream subjects use wildcard", func(t *testing.T) {
		subjects := reg.RSIGenerated.Stream.Subjects
		if len(subjects) != 1 || subjects[0] != "signal.events.>" {
			t.Errorf("expected [signal.events.>], got %v", subjects)
		}
	})

	t.Run("rsi latest query subject follows taxonomy", func(t *testing.T) {
		expected := "signal.query.rsi.latest"
		if reg.RSILatest.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.RSILatest.Subject)
		}
	})

	t.Run("rsi latest query queue group is signal.query", func(t *testing.T) {
		if reg.RSILatest.QueueGroup != "signal.query" {
			t.Errorf("expected signal.query, got %s", reg.RSILatest.QueueGroup)
		}
	})

	t.Run("rsi latest request type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSILatest.RequestType, ".v1.") {
			t.Errorf("expected versioned request type, got %s", reg.RSILatest.RequestType)
		}
	})

	t.Run("rsi latest reply type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSILatest.ReplyType, ".v1.") {
			t.Errorf("expected versioned reply type, got %s", reg.RSILatest.ReplyType)
		}
	})

	t.Run("all subjects are lowercase", func(t *testing.T) {
		for _, s := range []string{
			reg.RSIGenerated.Subject,
			reg.RSILatest.Subject,
		} {
			if s != strings.ToLower(s) {
				t.Errorf("subject must be lowercase: %s", s)
			}
		}
	})

	t.Run("stream max age is set", func(t *testing.T) {
		if reg.RSIGenerated.Stream.MaxAge <= 0 {
			t.Error("stream max age must be positive")
		}
	})

	t.Run("stream max bytes is set", func(t *testing.T) {
		if reg.RSIGenerated.Stream.MaxBytes <= 0 {
			t.Error("stream max bytes must be positive")
		}
	})
}

func TestSignalRegistry_LatestSpecByType(t *testing.T) {
	reg := adapternats.DefaultSignalRegistry()

	t.Run("rsi returns valid spec", func(t *testing.T) {
		spec, ok := reg.LatestSpecByType("rsi")
		if !ok {
			t.Fatal("expected rsi to be registered")
		}
		if spec.Subject != reg.RSILatest.Subject {
			t.Errorf("expected %s, got %s", reg.RSILatest.Subject, spec.Subject)
		}
		if spec.QueueGroup != reg.RSILatest.QueueGroup {
			t.Errorf("expected %s, got %s", reg.RSILatest.QueueGroup, spec.QueueGroup)
		}
	})

	t.Run("unknown type returns false", func(t *testing.T) {
		_, ok := reg.LatestSpecByType("macd")
		if ok {
			t.Error("expected unknown type to return false")
		}
	})

	t.Run("empty type returns false", func(t *testing.T) {
		_, ok := reg.LatestSpecByType("")
		if ok {
			t.Error("expected empty type to return false")
		}
	})
}

func TestStoreRSISignalConsumer(t *testing.T) {
	spec := adapternats.StoreRSISignalConsumer()

	t.Run("durable follows naming convention", func(t *testing.T) {
		if spec.Durable != "store-signal-rsi" {
			t.Errorf("expected store-signal-rsi, got %s", spec.Durable)
		}
	})

	t.Run("durable uses hyphens not underscores", func(t *testing.T) {
		if strings.Contains(spec.Durable, "_") {
			t.Errorf("durable name should use hyphens, not underscores: %s", spec.Durable)
		}
	})

	t.Run("subject uses wildcard for routing", func(t *testing.T) {
		if !strings.HasSuffix(spec.Event.Subject, ".>") {
			t.Errorf("consumer subject should end with .> for routing: %s", spec.Event.Subject)
		}
	})

	t.Run("stream name matches registry", func(t *testing.T) {
		if spec.Event.Stream.Name != "SIGNAL_EVENTS" {
			t.Errorf("expected SIGNAL_EVENTS, got %s", spec.Event.Stream.Name)
		}
	})

	t.Run("max deliver is bounded", func(t *testing.T) {
		if spec.MaxDeliver <= 0 {
			t.Error("max deliver must be positive")
		}
		if spec.MaxDeliver > 10 {
			t.Errorf("max deliver seems too high: %d", spec.MaxDeliver)
		}
	})

	t.Run("ack wait is positive", func(t *testing.T) {
		if spec.AckWait <= 0 {
			t.Error("ack wait must be positive")
		}
	})
}
