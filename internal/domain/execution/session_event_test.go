package execution

import (
	"testing"
)

func TestSessionLifecycleEventDeduplicationKey(t *testing.T) {
	t.Parallel()

	event := SessionLifecycleEvent{
		SessionID: "session_20260326_120000",
		Status:    SessionClosed,
	}

	key := event.DeduplicationKey()
	expected := "session-lifecycle:session_20260326_120000:closed"
	if key != expected {
		t.Fatalf("expected dedup key %q, got %q", expected, key)
	}
}

func TestSessionLifecycleEventDeduplicationKeyHalted(t *testing.T) {
	t.Parallel()

	event := SessionLifecycleEvent{
		SessionID: "session_20260326_120000",
		Status:    SessionHalted,
	}

	key := event.DeduplicationKey()
	expected := "session-lifecycle:session_20260326_120000:halted"
	if key != expected {
		t.Fatalf("expected dedup key %q, got %q", expected, key)
	}
}

func TestSessionLifecycleEventDeduplicationKeyUniqueness(t *testing.T) {
	t.Parallel()

	closed := SessionLifecycleEvent{SessionID: "session_20260326_120000", Status: SessionClosed}
	halted := SessionLifecycleEvent{SessionID: "session_20260326_120000", Status: SessionHalted}

	if closed.DeduplicationKey() == halted.DeduplicationKey() {
		t.Fatal("closed and halted events for the same session must have different dedup keys")
	}
}
