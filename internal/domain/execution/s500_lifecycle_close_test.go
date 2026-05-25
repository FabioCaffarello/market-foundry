package execution

import (
	"testing"
	"time"

	"internal/shared/clock"
)

// ---------------------------------------------------------------------------
// S500: Lifecycle close hardening — session edge case tests
// ---------------------------------------------------------------------------

// --- Double-close prevention (idempotency guard) ---

func TestSession_Close_AlreadyClosed_ReturnsProblem(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 10, Filled: 8, Rejected: 2},
	}

	// First close succeeds.
	if prob := s.Close(clock.SystemClock{}, counters); prob != nil {
		t.Fatalf("first Close() should succeed, got: %v", prob)
	}

	// Second close must fail.
	if prob := s.Close(clock.SystemClock{}, counters); prob == nil {
		t.Fatal("second Close() should return problem for already-terminal session")
	}
}

func TestSession_Halt_AlreadyHalted_ReturnsProblem(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 5, Filled: 3, Rejected: 1, Errors: 1},
	}

	if prob := s.Halt(clock.SystemClock{}, "kill-switch", counters); prob != nil {
		t.Fatalf("first Halt() should succeed, got: %v", prob)
	}

	if prob := s.Halt(clock.SystemClock{}, "second-reason", counters); prob == nil {
		t.Fatal("second Halt() should return problem for already-terminal session")
	}
}

func TestSession_Close_ThenHalt_ReturnsProblem(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if prob := s.Close(clock.SystemClock{}, nil); prob != nil {
		t.Fatalf("Close() should succeed: %v", prob)
	}
	if prob := s.Halt(clock.SystemClock{}, "late-halt", nil); prob == nil {
		t.Fatal("Halt() after Close() should return problem")
	}
}

func TestSession_Halt_ThenClose_ReturnsProblem(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if prob := s.Halt(clock.SystemClock{}, "reason", nil); prob != nil {
		t.Fatalf("Halt() should succeed: %v", prob)
	}
	if prob := s.Close(clock.SystemClock{}, nil); prob == nil {
		t.Fatal("Close() after Halt() should return problem")
	}
}

// --- Temporal ordering validation ---

func TestSession_Validate_ClosedAtBeforeStartedAt(t *testing.T) {
	started := time.Date(2026, 3, 28, 14, 0, 0, 0, time.UTC)
	closedBefore := time.Date(2026, 3, 28, 13, 0, 0, 0, time.UTC) // 1 hour before start

	s := Session{
		SessionID: "s1",
		Status:    SessionClosed,
		StartedAt: started,
		ClosedAt:  &closedBefore,
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	prob := s.Validate()
	if prob == nil {
		t.Fatal("Validate() should return problem when closed_at precedes started_at")
	}
}

func TestSession_Validate_ClosedAtEqualsStartedAt_IsValid(t *testing.T) {
	ts := time.Date(2026, 3, 28, 14, 0, 0, 0, time.UTC)
	s := Session{
		SessionID: "s1",
		Status:    SessionClosed,
		StartedAt: ts,
		ClosedAt:  &ts,
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if prob := s.Validate(); prob != nil {
		t.Fatalf("Validate() should accept closed_at == started_at, got: %v", prob)
	}
}

// --- InFlight counter ---

func TestSession_HasInFlightOrders(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionClosed,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
		SegmentCounters: []SessionSegmentCounters{
			{Segment: "spot", Processed: 10, Filled: 8, Rejected: 1, InFlight: 1},
		},
	}

	if !s.HasInFlightOrders() {
		t.Error("HasInFlightOrders() should return true when InFlight > 0")
	}
	if s.TotalInFlight() != 1 {
		t.Errorf("TotalInFlight() = %d, want 1", s.TotalInFlight())
	}
}

func TestSession_HasInFlightOrders_ZeroInFlight(t *testing.T) {
	s := Session{
		SegmentCounters: []SessionSegmentCounters{
			{Segment: "spot", Processed: 10, Filled: 10},
		},
	}

	if s.HasInFlightOrders() {
		t.Error("HasInFlightOrders() should return false when InFlight == 0")
	}
	if s.TotalInFlight() != 0 {
		t.Errorf("TotalInFlight() = %d, want 0", s.TotalInFlight())
	}
}

func TestSession_HasInFlightOrders_MultiSegment(t *testing.T) {
	s := Session{
		SegmentCounters: []SessionSegmentCounters{
			{Segment: "spot", Processed: 10, Filled: 10, InFlight: 0},
			{Segment: "futures", Processed: 5, Filled: 3, InFlight: 2},
		},
	}

	if !s.HasInFlightOrders() {
		t.Error("HasInFlightOrders() should detect in-flight in any segment")
	}
	if s.TotalInFlight() != 2 {
		t.Errorf("TotalInFlight() = %d, want 2", s.TotalInFlight())
	}
}

func TestSession_HasInFlightOrders_EmptyCounters(t *testing.T) {
	s := Session{}
	if s.HasInFlightOrders() {
		t.Error("HasInFlightOrders() should return false for empty counters")
	}
}

// --- Close preserves counters including InFlight ---

func TestSession_Close_PreservesInFlightCounter(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 10, Filled: 8, Rejected: 1, InFlight: 1},
	}

	if prob := s.Close(clock.SystemClock{}, counters); prob != nil {
		t.Fatalf("Close() failed: %v", prob)
	}

	if !s.HasInFlightOrders() {
		t.Error("InFlight counter should be preserved through Close()")
	}
	if s.SegmentCounters[0].InFlight != 1 {
		t.Errorf("InFlight = %d, want 1", s.SegmentCounters[0].InFlight)
	}
}

// --- Session status terminal state coverage ---

func TestSession_Close_FromOpen_Succeeds(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if prob := s.Close(clock.SystemClock{}, nil); prob != nil {
		t.Fatalf("Close from open should succeed: %v", prob)
	}
	if s.Status != SessionClosed {
		t.Errorf("status = %q, want closed", s.Status)
	}
}

func TestSession_Halt_FromOpen_Succeeds(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if prob := s.Halt(clock.SystemClock{}, "test-reason", nil); prob != nil {
		t.Fatalf("Halt from open should succeed: %v", prob)
	}
	if s.Status != SessionHalted {
		t.Errorf("status = %q, want halted", s.Status)
	}
	if s.HaltReason != "test-reason" {
		t.Errorf("halt_reason = %q, want test-reason", s.HaltReason)
	}
}
