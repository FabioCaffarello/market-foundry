package nats_test

import (
	"strings"
	"testing"

	adapternats "internal/adapters/nats"
)

func TestObservationRegistry_SubjectConventions(t *testing.T) {
	reg := adapternats.DefaultObservationRegistry()

	t.Run("trade subject follows taxonomy", func(t *testing.T) {
		expected := "observation.events.market.trade"
		if reg.TradeReceived.Subject != expected {
			t.Errorf("expected %s, got %s", expected, reg.TradeReceived.Subject)
		}
	})

	t.Run("trade type is versioned", func(t *testing.T) {
		if !strings.Contains(reg.TradeReceived.Type, ".v1.") {
			t.Errorf("expected versioned type, got %s", reg.TradeReceived.Type)
		}
	})

	t.Run("stream name is OBSERVATION_EVENTS", func(t *testing.T) {
		if reg.TradeReceived.Stream.Name != "OBSERVATION_EVENTS" {
			t.Errorf("expected OBSERVATION_EVENTS, got %s", reg.TradeReceived.Stream.Name)
		}
	})

	t.Run("stream subjects use wildcard", func(t *testing.T) {
		subjects := reg.TradeReceived.Stream.Subjects
		if len(subjects) != 1 || subjects[0] != "observation.events.market.>" {
			t.Errorf("expected [observation.events.market.>], got %v", subjects)
		}
	})

	t.Run("all subjects are lowercase", func(t *testing.T) {
		if reg.TradeReceived.Subject != strings.ToLower(reg.TradeReceived.Subject) {
			t.Errorf("subject must be lowercase: %s", reg.TradeReceived.Subject)
		}
	})
}

func TestDeriveObservationConsumer_Spec(t *testing.T) {
	spec := adapternats.DeriveObservationConsumer()

	if spec.Durable != "derive-observation" {
		t.Errorf("expected durable derive-observation, got %s", spec.Durable)
	}
	if spec.MaxDeliver < 1 {
		t.Error("max_deliver must be at least 1")
	}
}

func TestObservationRegistry_StreamConstraints(t *testing.T) {
	reg := adapternats.DefaultObservationRegistry()
	stream := reg.TradeReceived.Stream

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
}

func TestObservationRegistry_SubjectExtension(t *testing.T) {
	reg := adapternats.DefaultObservationRegistry()

	// Publisher extends the base subject with ".{source}".
	// The stream wildcard "observation.events.market.>" must match this.
	baseSubject := reg.TradeReceived.Subject
	extended := baseSubject + ".binancef"

	streamSubjects := reg.TradeReceived.Stream.Subjects
	if len(streamSubjects) == 0 {
		t.Fatal("stream must have at least one subject pattern")
	}

	// The wildcard "observation.events.market.>" should match "observation.events.market.trade.binancef"
	// This is a structural invariant — if the stream pattern doesn't cover extended subjects, messages are lost.
	wildcard := streamSubjects[0]
	if !strings.HasSuffix(wildcard, ".>") {
		t.Errorf("stream subject must use .> wildcard: %s", wildcard)
	}
	// The base of the wildcard must be a prefix of the extended subject.
	wildcardBase := strings.TrimSuffix(wildcard, ".>")
	if !strings.HasPrefix(extended, wildcardBase) {
		t.Errorf("extended subject %s must match stream wildcard %s", extended, wildcard)
	}
}

func TestDeriveObservationConsumer_FilterMatchesPublisher(t *testing.T) {
	spec := adapternats.DeriveObservationConsumer()
	reg := adapternats.DefaultObservationRegistry()

	// Consumer filter must match publisher's extended subjects.
	// Publisher: "observation.events.market.trade.{source}"
	// Consumer: "observation.events.market.trade.>"
	publisherBase := reg.TradeReceived.Subject
	consumerFilter := spec.Event.Subject

	filterBase := strings.TrimSuffix(consumerFilter, ".>")
	if filterBase != publisherBase {
		t.Errorf("consumer filter base %s must match publisher subject %s", filterBase, publisherBase)
	}
}

func TestDeriveObservationConsumer_AckWaitPositive(t *testing.T) {
	spec := adapternats.DeriveObservationConsumer()
	if spec.AckWait <= 0 {
		t.Errorf("AckWait must be positive, got %v", spec.AckWait)
	}
}

func TestDeriveObservationConsumer_MaxDeliverBounded(t *testing.T) {
	spec := adapternats.DeriveObservationConsumer()
	if spec.MaxDeliver < 1 || spec.MaxDeliver > 10 {
		t.Errorf("MaxDeliver should be between 1-10, got %d", spec.MaxDeliver)
	}
}

func TestObservationRegistry_TypeVersioning(t *testing.T) {
	reg := adapternats.DefaultObservationRegistry()
	eventType := reg.TradeReceived.Type

	// Type must contain a version marker.
	if !strings.Contains(eventType, ".v1.") {
		t.Errorf("event type must contain version marker .v1., got %s", eventType)
	}

	// Type must start with domain prefix.
	if !strings.HasPrefix(eventType, "observation.") {
		t.Errorf("event type must start with observation., got %s", eventType)
	}
}
