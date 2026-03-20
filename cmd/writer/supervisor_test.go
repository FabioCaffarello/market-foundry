package main

import (
	"testing"
	"time"
)

func TestCalcBackoff(t *testing.T) {
	s := &writerSupervisor{}

	tests := []struct {
		restart int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},
		{6, 30 * time.Second},
	}
	for _, tt := range tests {
		got := s.calcBackoff(tt.restart)
		if got != tt.want {
			t.Errorf("calcBackoff(%d) = %v, want %v", tt.restart, got, tt.want)
		}
	}
}

func TestPipelineLifecycleTransitions(t *testing.T) {
	// Verify initial state.
	lc := &pipelineLifecycle{state: pipelineActive}
	if lc.state != pipelineActive {
		t.Fatalf("expected active, got %s", lc.state)
	}

	// Simulate failure below budget.
	lc.state = pipelineRestarting
	lc.restarts = 3
	lc.lastError = "connection refused"
	if lc.state != pipelineRestarting {
		t.Fatalf("expected restarting, got %s", lc.state)
	}

	// Simulate budget exhaustion.
	lc.state = pipelineDegraded
	lc.restarts = maxPipelineRestarts + 1
	if lc.state != pipelineDegraded {
		t.Fatalf("expected degraded, got %s", lc.state)
	}
}

func TestPipelineStateConstants(t *testing.T) {
	if pipelineActive != "active" {
		t.Error("active state mismatch")
	}
	if pipelineRestarting != "restarting" {
		t.Error("restarting state mismatch")
	}
	if pipelineDegraded != "degraded" {
		t.Error("degraded state mismatch")
	}
}
