package execution

import (
	"testing"
	"time"

	"internal/shared/clock"
)

func TestNewSessionID(t *testing.T) {
	ts := time.Date(2026, 3, 24, 14, 42, 13, 0, time.UTC)
	id := NewSessionID(ts)
	expected := "session_20260324_144213"
	if id != expected {
		t.Errorf("NewSessionID(%v) = %q, want %q", ts, id, expected)
	}
}

func TestSessionValidate_Valid(t *testing.T) {
	now := time.Now().UTC()
	s := Session{
		SessionID: "session_20260324_144213",
		Status:    SessionOpen,
		StartedAt: now,
		Config: SessionConfigSnapshot{
			VenueType: "binance_spot_testnet",
		},
		Activation: SessionActivationSnapshot{
			Adapter:     AdapterVenue,
			Credentials: CredentialPresent,
			GateStatus:  GateActive,
			Effective:   ModeVenueLive,
		},
	}

	if prob := s.Validate(); prob != nil {
		t.Errorf("Validate() returned unexpected problem: %v", prob)
	}
}

func TestSessionValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		session Session
		field   string
	}{
		{
			name:    "missing session_id",
			session: Session{Status: SessionOpen, StartedAt: time.Now(), Config: SessionConfigSnapshot{VenueType: "paper_simulator"}},
			field:   "session_id",
		},
		{
			name:    "invalid status",
			session: Session{SessionID: "s1", Status: "bad", StartedAt: time.Now(), Config: SessionConfigSnapshot{VenueType: "paper_simulator"}},
			field:   "status",
		},
		{
			name:    "missing started_at",
			session: Session{SessionID: "s1", Status: SessionOpen, Config: SessionConfigSnapshot{VenueType: "paper_simulator"}},
			field:   "started_at",
		},
		{
			name:    "missing venue_type",
			session: Session{SessionID: "s1", Status: SessionOpen, StartedAt: time.Now(), Config: SessionConfigSnapshot{}},
			field:   "config.venue_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := tt.session.Validate()
			if prob == nil {
				t.Fatal("Validate() returned nil, expected problem")
			}
		})
	}
}

func TestSessionValidate_TerminalRequiresClosedAt(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionClosed,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	prob := s.Validate()
	if prob == nil {
		t.Fatal("Validate() returned nil, expected problem for terminal without closed_at")
	}
}

func TestSessionValidate_HaltedRequiresReason(t *testing.T) {
	now := time.Now().UTC()
	s := Session{
		SessionID: "s1",
		Status:    SessionHalted,
		StartedAt: now,
		ClosedAt:  &now,
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	prob := s.Validate()
	if prob == nil {
		t.Fatal("Validate() returned nil, expected problem for halted without reason")
	}
}

func TestSessionClose(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 10, Filled: 8, Rejected: 2},
	}

	if prob := s.Close(clock.SystemClock{}, counters); prob != nil {
		t.Fatalf("Close() returned unexpected problem: %v", prob)
	}

	if s.Status != SessionClosed {
		t.Errorf("Status = %q, want %q", s.Status, SessionClosed)
	}
	if s.ClosedAt == nil {
		t.Fatal("ClosedAt is nil after Close()")
	}
	if len(s.SegmentCounters) != 1 {
		t.Fatalf("SegmentCounters len = %d, want 1", len(s.SegmentCounters))
	}
	if s.SegmentCounters[0].Processed != 10 {
		t.Errorf("Processed = %d, want 10", s.SegmentCounters[0].Processed)
	}
}

func TestSessionHalt(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 5, Filled: 3, Rejected: 1, Errors: 1},
	}

	if prob := s.Halt(clock.SystemClock{}, "operator-kill-switch", counters); prob != nil {
		t.Fatalf("Halt() returned unexpected problem: %v", prob)
	}

	if s.Status != SessionHalted {
		t.Errorf("Status = %q, want %q", s.Status, SessionHalted)
	}
	if s.HaltReason != "operator-kill-switch" {
		t.Errorf("HaltReason = %q, want %q", s.HaltReason, "operator-kill-switch")
	}
	if s.ClosedAt == nil {
		t.Fatal("ClosedAt is nil after Halt()")
	}
}

func TestSessionDuration(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(15 * time.Minute)
	s := Session{
		SessionID: "s1",
		Status:    SessionClosed,
		StartedAt: now,
		ClosedAt:  &later,
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	d := s.Duration()
	if d != 15*time.Minute {
		t.Errorf("Duration() = %v, want %v", d, 15*time.Minute)
	}
}

func TestSessionDuration_Open(t *testing.T) {
	s := Session{
		SessionID: "s1",
		Status:    SessionOpen,
		StartedAt: time.Now().UTC(),
		Config:    SessionConfigSnapshot{VenueType: "paper_simulator"},
	}

	if d := s.Duration(); d != 0 {
		t.Errorf("Duration() = %v, want 0 for open session", d)
	}
}

func TestValidSessionStatus(t *testing.T) {
	tests := []struct {
		status SessionStatus
		valid  bool
	}{
		{SessionOpen, true},
		{SessionClosed, true},
		{SessionHalted, true},
		{"bad", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := ValidSessionStatus(tt.status); got != tt.valid {
			t.Errorf("ValidSessionStatus(%q) = %v, want %v", tt.status, got, tt.valid)
		}
	}
}

func TestSessionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		terminal bool
	}{
		{SessionOpen, false},
		{SessionClosed, true},
		{SessionHalted, true},
	}

	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.terminal {
			t.Errorf("(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}
