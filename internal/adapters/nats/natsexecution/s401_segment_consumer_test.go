package natsexecution

import (
	"strings"
	"testing"
)

// ── S401: Segment-scoped consumer spec tests ────────────────────────

func TestSegmentConsumerSingleSourceFilterSubject(t *testing.T) {
	spec := ExecuteVenueIntakeConsumerForSegments([]string{"binancef"})

	if len(spec.FilterSubjects) != 1 {
		t.Fatalf("expected 1 filter subject, got %d", len(spec.FilterSubjects))
	}
	want := "execution.events.paper_order.submitted.binancef.>"
	if spec.FilterSubjects[0] != want {
		t.Fatalf("expected filter subject %q, got %q", want, spec.FilterSubjects[0])
	}
}

func TestSegmentConsumerDualSourceFilterSubjects(t *testing.T) {
	spec := ExecuteVenueIntakeConsumerForSegments([]string{"binances", "binancef"})

	if len(spec.FilterSubjects) != 2 {
		t.Fatalf("expected 2 filter subjects, got %d", len(spec.FilterSubjects))
	}
	for _, sub := range spec.FilterSubjects {
		if !strings.HasPrefix(sub, "execution.events.paper_order.submitted.") {
			t.Fatalf("unexpected subject prefix: %q", sub)
		}
		if !strings.HasSuffix(sub, ".>") {
			t.Fatalf("subject must end with .> wildcard: %q", sub)
		}
	}
}

func TestSegmentConsumerEmptySourcesFallsBackToWildcard(t *testing.T) {
	spec := ExecuteVenueIntakeConsumerForSegments(nil)

	if len(spec.FilterSubjects) != 0 {
		t.Fatalf("expected no FilterSubjects for empty sources, got %d", len(spec.FilterSubjects))
	}
	// Falls back to Event.Subject (the default wildcard).
	want := "execution.events.paper_order.submitted.>"
	if spec.Event.Subject != want {
		t.Fatalf("expected Event.Subject %q, got %q", want, spec.Event.Subject)
	}
}

func TestSegmentConsumerDurableNamePreserved(t *testing.T) {
	spec := ExecuteVenueIntakeConsumerForSegments([]string{"binancef"})
	if spec.Durable != "execute-venue-market-order-intake" {
		t.Fatalf("expected durable name preserved, got %q", spec.Durable)
	}
}

func TestSegmentConsumerFilterSubjectsContainSourcePrefix(t *testing.T) {
	sources := []string{"binances", "binancef"}
	spec := ExecuteVenueIntakeConsumerForSegments(sources)

	for i, src := range sources {
		if !strings.Contains(spec.FilterSubjects[i], "."+src+".") {
			t.Fatalf("filter subject %q does not contain source %q", spec.FilterSubjects[i], src)
		}
	}
}

func TestSegmentConsumerSubjectIsolation(t *testing.T) {
	// A spot-only consumer must NOT match futures subjects.
	spotSpec := ExecuteVenueIntakeConsumerForSegments([]string{"binances"})

	for _, sub := range spotSpec.FilterSubjects {
		if strings.Contains(sub, "binancef") {
			t.Fatalf("spot-only consumer has futures subject: %q", sub)
		}
	}

	// A futures-only consumer must NOT match spot subjects.
	futuresSpec := ExecuteVenueIntakeConsumerForSegments([]string{"binancef"})
	for _, sub := range futuresSpec.FilterSubjects {
		if strings.Contains(sub, "binances") {
			t.Fatalf("futures-only consumer has spot subject: %q", sub)
		}
	}
}
