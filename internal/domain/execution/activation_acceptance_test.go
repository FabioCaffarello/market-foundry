package execution_test

import (
	"testing"
	"time"

	"internal/domain/execution"
)

// ---------- S340: Activation Acceptance Scenarios ----------
//
// These tests validate the three canonical acceptance scenarios for venue activation:
//   AC-1: Inactive → Active (off→on)
//   AC-2: Active → Halt (on→halt)
//   AC-3: Halt → Rollback (controlled return to paper)
//
// Each test simulates a state transition by constructing sequential ActivationSurface
// snapshots and asserting the effective mode and safety predicates at each step.

func TestActivationAcceptance_InactiveToActive(t *testing.T) {
	// AC-1: off→on
	// Venue adapter loaded, gate halted (safe default), credentials present.
	// Operator sets gate to active → venue_live.

	// Step 1: gate halted — venue is reachable but not live.
	gate1 := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "initial-deploy",
		UpdatedBy: "deploy-pipeline",
		UpdatedAt: time.Now().UTC(),
	}
	s1 := execution.NewActivationSurface(execution.AdapterVenue, gate1, execution.CredentialPresent)

	t.Logf("[AC-1/step-1] effective=%s is_live=%v can_reach_venue=%v", s1.Effective, s1.IsLive(), s1.CanReachVenue())

	if s1.Effective != execution.ModeVenueHalted {
		t.Fatalf("[AC-1/step-1] want effective=venue_halted, got %s", s1.Effective)
	}
	if s1.IsLive() {
		t.Fatalf("[AC-1/step-1] want IsLive=false, got true")
	}
	if !s1.CanReachVenue() {
		t.Fatalf("[AC-1/step-1] want CanReachVenue=true, got false")
	}

	// Step 2: operator opens the gate → venue_live.
	gate2 := execution.ControlGate{
		Status:    execution.GateActive,
		Reason:    "smoke-passed",
		UpdatedBy: "operator",
		UpdatedAt: time.Now().UTC(),
	}
	s2 := execution.NewActivationSurface(execution.AdapterVenue, gate2, execution.CredentialPresent)

	t.Logf("[AC-1/step-2] effective=%s is_live=%v can_reach_venue=%v", s2.Effective, s2.IsLive(), s2.CanReachVenue())

	if s2.Effective != execution.ModeVenueLive {
		t.Fatalf("[AC-1/step-2] want effective=venue_live, got %s", s2.Effective)
	}
	if !s2.IsLive() {
		t.Fatalf("[AC-1/step-2] want IsLive=true, got false")
	}
	if !s2.CanReachVenue() {
		t.Fatalf("[AC-1/step-2] want CanReachVenue=true, got false")
	}
}

func TestActivationAcceptance_ActiveToHalt(t *testing.T) {
	// AC-2: on→halt
	// Venue is live. Operator halts the gate. Venue becomes halted, IsLive goes false.

	// Step 1: venue_live.
	gate1 := execution.ControlGate{
		Status:    execution.GateActive,
		Reason:    "normal-operation",
		UpdatedBy: "operator",
		UpdatedAt: time.Now().UTC(),
	}
	s1 := execution.NewActivationSurface(execution.AdapterVenue, gate1, execution.CredentialPresent)

	t.Logf("[AC-2/step-1] effective=%s is_live=%v", s1.Effective, s1.IsLive())

	if s1.Effective != execution.ModeVenueLive {
		t.Fatalf("[AC-2/step-1] want effective=venue_live, got %s", s1.Effective)
	}
	if !s1.IsLive() {
		t.Fatalf("[AC-2/step-1] want IsLive=true, got false")
	}

	// Step 2: operator halts the gate.
	haltTime := time.Now().UTC()
	gate2 := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "operator-halt",
		UpdatedBy: "smoke-s340",
		UpdatedAt: haltTime,
	}
	s2 := execution.NewActivationSurface(execution.AdapterVenue, gate2, execution.CredentialPresent)

	t.Logf("[AC-2/step-2] effective=%s is_live=%v reason=%q updated_by=%q",
		s2.Effective, s2.IsLive(), s2.Gate.Reason, s2.Gate.UpdatedBy)

	if s2.Effective != execution.ModeVenueHalted {
		t.Fatalf("[AC-2/step-2] want effective=venue_halted, got %s", s2.Effective)
	}
	if s2.IsLive() {
		t.Fatalf("[AC-2/step-2] want IsLive=false, got true")
	}

	// Verify audit fields are preserved.
	if s2.Gate.Reason != "operator-halt" {
		t.Fatalf("[AC-2/step-2] want gate.Reason=operator-halt, got %q", s2.Gate.Reason)
	}
	if s2.Gate.UpdatedBy != "smoke-s340" {
		t.Fatalf("[AC-2/step-2] want gate.UpdatedBy=smoke-s340, got %q", s2.Gate.UpdatedBy)
	}
	if !s2.Gate.UpdatedAt.Equal(haltTime) {
		t.Fatalf("[AC-2/step-2] want gate.UpdatedAt=%v, got %v", haltTime, s2.Gate.UpdatedAt)
	}
}

func TestActivationAcceptance_HaltToRollback(t *testing.T) {
	// AC-3: halt→rollback
	// Venue is halted. Operator rolls back: stop binary, change config to paper, restart.
	// This simulates the full rollback: halt gate → stop binary → change config → restart.

	// Step 1: venue_halted (starting state after AC-2 halt).
	gate1 := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "pre-rollback",
		UpdatedBy: "operator",
		UpdatedAt: time.Now().UTC(),
	}
	s1 := execution.NewActivationSurface(execution.AdapterVenue, gate1, execution.CredentialPresent)

	t.Logf("[AC-3/step-1] effective=%s is_live=%v can_reach_venue=%v", s1.Effective, s1.IsLive(), s1.CanReachVenue())

	if s1.Effective != execution.ModeVenueHalted {
		t.Fatalf("[AC-3/step-1] want effective=venue_halted, got %s", s1.Effective)
	}

	// Step 2: binary restarted with paper config, no credentials.
	gate2 := execution.ControlGate{
		Status:    execution.GateActive,
		UpdatedAt: time.Now().UTC(),
	}
	s2 := execution.NewActivationSurface(execution.AdapterPaper, gate2, execution.CredentialAbsent)

	t.Logf("[AC-3/step-2] effective=%s is_live=%v can_reach_venue=%v", s2.Effective, s2.IsLive(), s2.CanReachVenue())

	if s2.Effective != execution.ModePaper {
		t.Fatalf("[AC-3/step-2] want effective=paper, got %s", s2.Effective)
	}
	if s2.IsLive() {
		t.Fatalf("[AC-3/step-2] want IsLive=false, got true")
	}
	if s2.CanReachVenue() {
		t.Fatalf("[AC-3/step-2] want CanReachVenue=false, got true")
	}
}

func TestActivationAcceptance_FullCycle(t *testing.T) {
	// AC-4: Full round trip — paper → venue_halted → venue_live → venue_halted → paper.
	// This is the canonical lifecycle acceptance test.

	type step struct {
		label   string
		adapter execution.AdapterState
		gate    execution.GateStatus
		creds   execution.CredentialState
		want    execution.EffectiveMode
	}

	steps := []step{
		{
			label:   "paper-baseline",
			adapter: execution.AdapterPaper,
			gate:    execution.GateActive,
			creds:   execution.CredentialAbsent,
			want:    execution.ModePaper,
		},
		{
			label:   "venue-deploy-halted",
			adapter: execution.AdapterVenue,
			gate:    execution.GateHalted,
			creds:   execution.CredentialPresent,
			want:    execution.ModeVenueHalted,
		},
		{
			label:   "gate-open-live",
			adapter: execution.AdapterVenue,
			gate:    execution.GateActive,
			creds:   execution.CredentialPresent,
			want:    execution.ModeVenueLive,
		},
		{
			label:   "gate-halt",
			adapter: execution.AdapterVenue,
			gate:    execution.GateHalted,
			creds:   execution.CredentialPresent,
			want:    execution.ModeVenueHalted,
		},
		{
			label:   "rollback-to-paper",
			adapter: execution.AdapterPaper,
			gate:    execution.GateActive,
			creds:   execution.CredentialAbsent,
			want:    execution.ModePaper,
		},
	}

	for i, s := range steps {
		gate := execution.ControlGate{
			Status:    s.gate,
			Reason:    s.label,
			UpdatedBy: "acceptance-s340",
			UpdatedAt: time.Now().UTC(),
		}
		surface := execution.NewActivationSurface(s.adapter, gate, s.creds)

		t.Logf("[AC-4/step-%d/%s] effective=%s is_live=%v can_reach_venue=%v",
			i+1, s.label, surface.Effective, surface.IsLive(), surface.CanReachVenue())

		if surface.Effective != s.want {
			t.Fatalf("[AC-4/step-%d/%s] want effective=%s, got %s", i+1, s.label, s.want, surface.Effective)
		}
	}
}

func TestActivationAcceptance_DegradedIsNotLive(t *testing.T) {
	// AC-5: Degraded mode (venue adapter, gate active, credentials absent) must never be live.
	gate := execution.ControlGate{
		Status:    execution.GateActive,
		UpdatedAt: time.Now().UTC(),
	}
	s := execution.NewActivationSurface(execution.AdapterVenue, gate, execution.CredentialAbsent)

	t.Logf("[AC-5] effective=%s is_live=%v can_reach_venue=%v", s.Effective, s.IsLive(), s.CanReachVenue())

	if s.Effective != execution.ModeVenueDegraded {
		t.Fatalf("[AC-5] want effective=venue_degraded, got %s", s.Effective)
	}
	if s.IsLive() {
		t.Fatalf("[AC-5] degraded mode must not be live, got IsLive=true")
	}
}

func TestActivationAcceptance_GateAuditFieldsSurviveTransition(t *testing.T) {
	// AC-6: Gate audit fields (Reason, UpdatedBy, UpdatedAt) must be accessible
	// through the ActivationSurface after construction.

	auditTime := time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC)
	gate := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "circuit-breaker-triggered",
		UpdatedBy: "monitoring-agent",
		UpdatedAt: auditTime,
	}

	s := execution.NewActivationSurface(execution.AdapterVenue, gate, execution.CredentialPresent)

	t.Logf("[AC-6] gate.Status=%s gate.Reason=%q gate.UpdatedBy=%q gate.UpdatedAt=%v",
		s.Gate.Status, s.Gate.Reason, s.Gate.UpdatedBy, s.Gate.UpdatedAt)

	if s.Gate.Status != execution.GateHalted {
		t.Fatalf("[AC-6] want gate.Status=halted, got %q", s.Gate.Status)
	}
	if s.Gate.Reason != "circuit-breaker-triggered" {
		t.Fatalf("[AC-6] want gate.Reason=circuit-breaker-triggered, got %q", s.Gate.Reason)
	}
	if s.Gate.UpdatedBy != "monitoring-agent" {
		t.Fatalf("[AC-6] want gate.UpdatedBy=monitoring-agent, got %q", s.Gate.UpdatedBy)
	}
	if !s.Gate.UpdatedAt.Equal(auditTime) {
		t.Fatalf("[AC-6] want gate.UpdatedAt=%v, got %v", auditTime, s.Gate.UpdatedAt)
	}
	if s.Gate.IsHalted() != true {
		t.Fatalf("[AC-6] want gate.IsHalted=true, got false")
	}
}
