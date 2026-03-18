package nats_test

import (
	"fmt"
	"strings"
	"testing"

	adapternats "internal/adapters/nats"
)

func TestDecisionRegistry_SubjectConventions(t *testing.T) {
	reg := adapternats.DefaultDecisionRegistry()

	t.Run("rsi oversold evaluated subject follows taxonomy", func(t *testing.T) {
		expected := "decision.events.rsi_oversold.evaluated"
		if reg.RSIOversoldEvaluated.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.RSIOversoldEvaluated.Subject)
		}
	})

	t.Run("rsi oversold type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSIOversoldEvaluated.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.RSIOversoldEvaluated.Type)
		}
	})

	t.Run("stream name is DECISION_EVENTS", func(t *testing.T) {
		if reg.RSIOversoldEvaluated.Stream.Name != "DECISION_EVENTS" {
			t.Errorf("expected DECISION_EVENTS, got %s", reg.RSIOversoldEvaluated.Stream.Name)
		}
	})

	t.Run("stream subjects use wildcard", func(t *testing.T) {
		subjects := reg.RSIOversoldEvaluated.Stream.Subjects
		if len(subjects) != 1 || subjects[0] != "decision.events.>" {
			t.Errorf("expected [decision.events.>], got %v", subjects)
		}
	})

	t.Run("rsi oversold latest query subject follows taxonomy", func(t *testing.T) {
		expected := "decision.query.rsi_oversold.latest"
		if reg.RSIOversoldLatest.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.RSIOversoldLatest.Subject)
		}
	})

	t.Run("rsi oversold latest query queue group is decision.query", func(t *testing.T) {
		if reg.RSIOversoldLatest.QueueGroup != "decision.query" {
			t.Errorf("expected decision.query, got %s", reg.RSIOversoldLatest.QueueGroup)
		}
	})

	t.Run("rsi oversold latest request type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSIOversoldLatest.RequestType, ".v1.") {
			t.Errorf("expected versioned request type, got %s", reg.RSIOversoldLatest.RequestType)
		}
	})

	t.Run("rsi oversold latest reply type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.RSIOversoldLatest.ReplyType, ".v1.") {
			t.Errorf("expected versioned reply type, got %s", reg.RSIOversoldLatest.ReplyType)
		}
	})

	t.Run("all subjects are lowercase", func(t *testing.T) {
		for _, s := range []string{
			reg.RSIOversoldEvaluated.Subject,
			reg.RSIOversoldLatest.Subject,
		} {
			if s != strings.ToLower(s) {
				t.Errorf("subject must be lowercase: %s", s)
			}
		}
	})

	t.Run("stream max age is set", func(t *testing.T) {
		if reg.RSIOversoldEvaluated.Stream.MaxAge <= 0 {
			t.Error("stream max age must be positive")
		}
	})

	t.Run("stream max bytes is set", func(t *testing.T) {
		if reg.RSIOversoldEvaluated.Stream.MaxBytes <= 0 {
			t.Error("stream max bytes must be positive")
		}
	})
}

func TestDecisionRegistry_LatestSpecByType(t *testing.T) {
	reg := adapternats.DefaultDecisionRegistry()

	t.Run("rsi_oversold returns valid spec", func(t *testing.T) {
		spec, ok := reg.LatestSpecByType("rsi_oversold")
		if !ok {
			t.Fatal("expected rsi_oversold to be registered")
		}
		if spec.Subject != reg.RSIOversoldLatest.Subject {
			t.Errorf("expected %s, got %s", reg.RSIOversoldLatest.Subject, spec.Subject)
		}
		if spec.QueueGroup != reg.RSIOversoldLatest.QueueGroup {
			t.Errorf("expected %s, got %s", reg.RSIOversoldLatest.QueueGroup, spec.QueueGroup)
		}
	})

	t.Run("unknown type returns false", func(t *testing.T) {
		_, ok := reg.LatestSpecByType("macd_crossover")
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

	t.Run("signal type is not a decision type", func(t *testing.T) {
		_, ok := reg.LatestSpecByType("rsi")
		if ok {
			t.Error("signal type 'rsi' should not be registered as a decision type")
		}
	})
}

func TestStoreRSIOversoldDecisionConsumer(t *testing.T) {
	spec := adapternats.StoreRSIOversoldDecisionConsumer()

	t.Run("durable follows naming convention", func(t *testing.T) {
		if spec.Durable != "store-decision-rsi-oversold" {
			t.Errorf("expected store-decision-rsi-oversold, got %s", spec.Durable)
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
		if spec.Event.Stream.Name != "DECISION_EVENTS" {
			t.Errorf("expected DECISION_EVENTS, got %s", spec.Event.Stream.Name)
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

	t.Run("subject includes decision type for filtering", func(t *testing.T) {
		if !strings.Contains(spec.Event.Subject, "rsi_oversold") {
			t.Errorf("consumer subject should filter by decision type: %s", spec.Event.Subject)
		}
	})
}

func TestDecisionRegistry_SubjectRoutingMultiSymbol(t *testing.T) {
	reg := adapternats.DefaultDecisionRegistry()
	base := reg.RSIOversoldEvaluated.Subject

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	sources := []string{"binancef"}

	subjects := make(map[string]bool)
	for _, src := range sources {
		for _, sym := range symbols {
			for _, tf := range timeframes {
				subj := fmt.Sprintf("%s.%s.%s.%d", base, src, sym, tf)
				if subjects[subj] {
					t.Fatalf("subject collision: %s", subj)
				}
				subjects[subj] = true
			}
		}
	}

	expectedCount := len(sources) * len(symbols) * len(timeframes)
	if len(subjects) != expectedCount {
		t.Errorf("expected %d unique subjects, got %d", expectedCount, len(subjects))
	}

	// Verify all subjects match the stream wildcard.
	for subj := range subjects {
		if !strings.HasPrefix(subj, "decision.events.") {
			t.Errorf("subject %s does not match stream wildcard decision.events.>", subj)
		}
	}

	// Verify consumer filter matches all subjects.
	consumerSpec := adapternats.StoreRSIOversoldDecisionConsumer()
	filterBase := strings.TrimSuffix(consumerSpec.Event.Subject, ".>")
	for subj := range subjects {
		if !strings.HasPrefix(subj, filterBase) {
			t.Errorf("subject %s does not match consumer filter %s", subj, consumerSpec.Event.Subject)
		}
	}
}
