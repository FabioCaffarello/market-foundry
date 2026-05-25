package execution

import (
	"testing"
	"time"

	domainexec "internal/domain/execution"
	"internal/shared/clock"
)

// S460: Canonical session metadata model validation.
// These tests verify that the session entity captures all required metadata
// for operational accountability and audit trail.

func TestS460_SessionEntityHasRequiredFields(t *testing.T) {
	now := time.Now().UTC()
	s := domainexec.Session{
		SessionID: domainexec.NewSessionID(now),
		Operator:  "fabio",
		Status:    domainexec.SessionOpen,
		StartedAt: now,
		Config: domainexec.SessionConfigSnapshot{
			VenueType: "binance_spot_testnet",
			DryRun:    true,
			Segments:  []string{"binances"},
		},
		Activation: domainexec.SessionActivationSnapshot{
			Adapter:     domainexec.AdapterVenue,
			Credentials: domainexec.CredentialPresent,
			GateStatus:  domainexec.GateActive,
			Effective:   domainexec.ModeVenueLive,
		},
		Artifacts: map[string]string{
			"config_file":  "deploy/configs/execute.jsonc",
			"compose_file": "deploy/compose/docker-compose.yaml",
		},
	}

	if prob := s.Validate(); prob != nil {
		t.Fatalf("valid session failed validation: %v", prob)
	}

	if s.SessionID == "" {
		t.Error("session_id must be set")
	}
	if s.Operator == "" {
		t.Error("operator must be set")
	}
	if s.Config.VenueType == "" {
		t.Error("config.venue_type must be set")
	}
	if s.Activation.Adapter == "" {
		t.Error("activation.adapter must be set")
	}
	if s.Activation.Effective == "" {
		t.Error("activation.effective must be set")
	}
}

func TestS460_SessionLifecycleTransitions(t *testing.T) {
	now := time.Now().UTC()
	s := domainexec.Session{
		SessionID: domainexec.NewSessionID(now),
		Operator:  "fabio",
		Status:    domainexec.SessionOpen,
		StartedAt: now,
		Config: domainexec.SessionConfigSnapshot{
			VenueType: "paper_simulator",
		},
		Activation: domainexec.SessionActivationSnapshot{
			Adapter:     domainexec.AdapterPaper,
			Credentials: domainexec.CredentialAbsent,
			GateStatus:  domainexec.GateActive,
			Effective:   domainexec.ModePaper,
		},
	}

	// Session starts as open.
	if s.Status != domainexec.SessionOpen {
		t.Fatalf("initial status = %q, want %q", s.Status, domainexec.SessionOpen)
	}
	if s.Status.IsTerminal() {
		t.Fatal("open session should not be terminal")
	}

	// Close with counters.
	counters := []domainexec.SessionSegmentCounters{
		{Segment: "spot", Processed: 42, Filled: 38, Rejected: 4},
	}
	if prob := s.Close(clock.SystemClock{}, counters); prob != nil {
		t.Fatalf("Close() returned unexpected problem: %v", prob)
	}

	if s.Status != domainexec.SessionClosed {
		t.Errorf("after Close, status = %q, want %q", s.Status, domainexec.SessionClosed)
	}
	if !s.Status.IsTerminal() {
		t.Error("closed session should be terminal")
	}
	if s.ClosedAt == nil {
		t.Error("closed_at must be set after Close()")
	}
	if s.Duration() == 0 {
		t.Error("duration should be positive after close")
	}
	if prob := s.Validate(); prob != nil {
		t.Errorf("closed session failed validation: %v", prob)
	}
}

func TestS460_SessionHaltCapturesReason(t *testing.T) {
	now := time.Now().UTC()
	s := domainexec.Session{
		SessionID: domainexec.NewSessionID(now),
		Status:    domainexec.SessionOpen,
		StartedAt: now,
		Config:    domainexec.SessionConfigSnapshot{VenueType: "binance_futures_testnet"},
		Activation: domainexec.SessionActivationSnapshot{
			Adapter:     domainexec.AdapterVenue,
			Credentials: domainexec.CredentialPresent,
			GateStatus:  domainexec.GateActive,
			Effective:   domainexec.ModeVenueLive,
		},
	}

	s.Halt(clock.SystemClock{}, "operator-kill-switch", []domainexec.SessionSegmentCounters{
		{Segment: "futures", Processed: 10, Filled: 7, Rejected: 2, Errors: 1},
	})

	if s.Status != domainexec.SessionHalted {
		t.Errorf("status = %q, want %q", s.Status, domainexec.SessionHalted)
	}
	if s.HaltReason != "operator-kill-switch" {
		t.Errorf("halt_reason = %q, want %q", s.HaltReason, "operator-kill-switch")
	}
	if prob := s.Validate(); prob != nil {
		t.Errorf("halted session failed validation: %v", prob)
	}
}

func TestS460_SessionConfigSnapshotPreservesState(t *testing.T) {
	snap := domainexec.SessionConfigSnapshot{
		VenueType:  "binance_spot_testnet",
		DryRun:     false,
		Segments:   []string{"binances", "binancef"},
		ConfigFile: "deploy/configs/execute-unified.jsonc",
	}

	if snap.VenueType != "binance_spot_testnet" {
		t.Errorf("VenueType = %q", snap.VenueType)
	}
	if snap.DryRun {
		t.Error("DryRun should be false")
	}
	if len(snap.Segments) != 2 {
		t.Errorf("Segments len = %d, want 2", len(snap.Segments))
	}
}

func TestS460_SessionActivationSnapshotCapturesSurface(t *testing.T) {
	snap := domainexec.SessionActivationSnapshot{
		Adapter:     domainexec.AdapterVenue,
		Credentials: domainexec.CredentialPresent,
		GateStatus:  domainexec.GateActive,
		Effective:   domainexec.ModeVenueLive,
	}

	if snap.Effective != domainexec.ModeVenueLive {
		t.Errorf("Effective = %q, want %q", snap.Effective, domainexec.ModeVenueLive)
	}
}

func TestS460_SessionSegmentCountersPerSegment(t *testing.T) {
	now := time.Now().UTC()
	s := domainexec.Session{
		SessionID: domainexec.NewSessionID(now),
		Status:    domainexec.SessionOpen,
		StartedAt: now,
		Config:    domainexec.SessionConfigSnapshot{VenueType: "paper_simulator"},
		Activation: domainexec.SessionActivationSnapshot{
			Adapter:   domainexec.AdapterPaper,
			Effective: domainexec.ModePaper,
		},
	}

	s.Close(clock.SystemClock{}, []domainexec.SessionSegmentCounters{
		{Segment: "spot", Processed: 100, Filled: 90, Rejected: 10},
		{Segment: "futures", Processed: 50, Filled: 45, Rejected: 5},
	})

	if len(s.SegmentCounters) != 2 {
		t.Fatalf("SegmentCounters len = %d, want 2", len(s.SegmentCounters))
	}

	totalProcessed := int64(0)
	for _, c := range s.SegmentCounters {
		totalProcessed += c.Processed
	}
	if totalProcessed != 150 {
		t.Errorf("total processed = %d, want 150", totalProcessed)
	}
}
