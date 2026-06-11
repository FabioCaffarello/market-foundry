package monitoring_test

import (
	"testing"
	"time"

	"internal/domain/execution"
	"internal/domain/monitoring"
)

func TestNewSessionSummary(t *testing.T) {
	now := time.Now().UTC()
	closed := now.Add(30 * time.Minute)
	session := execution.Session{
		SessionID: "session_20260326_120000",
		Operator:  "testuser",
		Status:    execution.SessionClosed,
		StartedAt: now,
		ClosedAt:  &closed,
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance",
			DryRun:    true,
			Segments:  []string{"spot"},
		},
		SegmentCounters: []execution.SessionSegmentCounters{
			{Segment: "spot", Processed: 100, Filled: 10, Rejected: 2, Errors: 1},
		},
	}

	summary := monitoring.NewSessionSummary(session)

	if summary.SessionID != session.SessionID {
		t.Errorf("SessionID = %q, want %q", summary.SessionID, session.SessionID)
	}
	if summary.Status != execution.SessionClosed {
		t.Errorf("Status = %q, want %q", summary.Status, execution.SessionClosed)
	}
	if summary.Duration == "" {
		t.Error("Duration should be set for closed session")
	}
	if len(summary.Counters) != 1 {
		t.Fatalf("Counters len = %d, want 1", len(summary.Counters))
	}
	if summary.Counters[0].Processed != 100 {
		t.Errorf("Counters[0].Processed = %d, want 100", summary.Counters[0].Processed)
	}
}

func TestNewSessionSummary_OpenSession(t *testing.T) {
	now := time.Now().UTC()
	session := execution.Session{
		SessionID: "session_20260326_120000",
		Status:    execution.SessionOpen,
		StartedAt: now,
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance",
			DryRun:    false,
		},
	}

	summary := monitoring.NewSessionSummary(session)

	if summary.Duration != "" {
		t.Errorf("Duration should be empty for open session, got %q", summary.Duration)
	}
	if summary.ClosedAt != nil {
		t.Error("ClosedAt should be nil for open session")
	}
}

func TestSurfaceAvailability_DegradedFamilies(t *testing.T) {
	tests := []struct {
		name     string
		surfaces monitoring.SurfaceAvailability
		want     int
	}{
		{
			name: "all available",
			surfaces: monitoring.SurfaceAvailability{
				Evidence: true, Signal: true, Decision: true,
				Strategy: true, Risk: true, Execution: true,
				Session: true, Analytical: true, Activation: true,
			},
			want: 0,
		},
		{
			name:     "none available",
			surfaces: monitoring.SurfaceAvailability{},
			want:     9,
		},
		{
			name: "partial",
			surfaces: monitoring.SurfaceAvailability{
				Evidence: true, Execution: true, Session: true,
			},
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			degraded := tt.surfaces.DegradedFamilies()
			if len(degraded) != tt.want {
				t.Errorf("DegradedFamilies() len = %d, want %d; degraded = %v", len(degraded), tt.want, degraded)
			}
		})
	}
}
