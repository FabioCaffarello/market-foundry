package monitoringclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/executionclient"
	"internal/application/monitoringclient"
	"internal/domain/execution"
	"internal/domain/monitoring"
	"internal/shared/clock"
	"internal/shared/problem"
)

// --- stubs ---

type stubSessionLister struct {
	sessions []execution.Session
	prob     *problem.Problem
}

func (s *stubSessionLister) Execute(_ context.Context, _ executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem) {
	if s.prob != nil {
		return executionclient.SessionListReply{}, s.prob
	}
	return executionclient.SessionListReply{Sessions: s.sessions, Total: len(s.sessions)}, nil
}

type stubGateReader struct {
	gate execution.ControlGate
	prob *problem.Problem
}

func (s *stubGateReader) Execute(_ context.Context, _ executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem) {
	if s.prob != nil {
		return executionclient.ExecutionControlReply{}, s.prob
	}
	return executionclient.ExecutionControlReply{Gate: s.gate}, nil
}

// --- tests ---

func TestGetOperationalState_FullWiring(t *testing.T) {
	now := time.Now().UTC()
	sessions := []execution.Session{
		{
			SessionID: "session_20260326_120000",
			Operator:  "op1",
			Status:    execution.SessionOpen,
			StartedAt: now,
			Config:    execution.SessionConfigSnapshot{VenueType: "binance", DryRun: true, Segments: []string{"spot"}},
		},
	}

	gate := execution.ControlGate{
		Status:    execution.GateActive,
		UpdatedAt: now,
	}

	surfaces := monitoring.SurfaceAvailability{
		Evidence: true, Signal: true, Decision: true,
		Strategy: true, Risk: true, Execution: true,
		Session: true, Analytical: true, Activation: true,
	}

	uc := monitoringclient.NewGetOperationalStateUseCase(
		&stubSessionLister{sessions: sessions},
		&stubGateReader{gate: gate},
		surfaces,
	)

	reply, prob := uc.Execute(context.Background(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	state := reply.State
	if state.Session == nil {
		t.Fatal("Session should not be nil")
	}
	if state.Session.SessionID != "session_20260326_120000" {
		t.Errorf("SessionID = %q, want session_20260326_120000", state.Session.SessionID)
	}
	if state.Gate == nil {
		t.Fatal("Gate should not be nil")
	}
	if state.Gate.Status != "active" {
		t.Errorf("Gate.Status = %q, want active", state.Gate.Status)
	}
	if !state.Surfaces.Evidence {
		t.Error("Surfaces.Evidence should be true")
	}
	if state.ObservedAt.IsZero() {
		t.Error("ObservedAt should be set")
	}
}

func TestGetOperationalState_NoSessions(t *testing.T) {
	uc := monitoringclient.NewGetOperationalStateUseCase(
		&stubSessionLister{sessions: nil},
		&stubGateReader{gate: execution.DefaultControlGate(clock.SystemClock{})},
		monitoring.SurfaceAvailability{},
	)

	reply, prob := uc.Execute(context.Background(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.State.Session != nil {
		t.Error("Session should be nil when no sessions exist")
	}
}

func TestGetOperationalState_NilDependencies(t *testing.T) {
	uc := monitoringclient.NewGetOperationalStateUseCase(nil, nil, monitoring.SurfaceAvailability{})

	reply, prob := uc.Execute(context.Background(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.State.Session != nil {
		t.Error("Session should be nil when lister is nil")
	}
	if reply.State.Gate != nil {
		t.Error("Gate should be nil when reader is nil")
	}
}

func TestGetOperationalState_SessionListerError(t *testing.T) {
	uc := monitoringclient.NewGetOperationalStateUseCase(
		&stubSessionLister{prob: problem.New(problem.Unavailable, "down")},
		&stubGateReader{gate: execution.DefaultControlGate(clock.SystemClock{})},
		monitoring.SurfaceAvailability{},
	)

	reply, prob := uc.Execute(context.Background(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	// Session unavailability is gracefully degraded, not an error.
	if reply.State.Session != nil {
		t.Error("Session should be nil on lister error")
	}
	if reply.State.Gate == nil {
		t.Error("Gate should still be populated despite session error")
	}
}

func TestGetOperationalState_GateHalted(t *testing.T) {
	now := time.Now().UTC()
	gate := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "emergency stop",
		UpdatedAt: now,
		UpdatedBy: "operator",
	}

	uc := monitoringclient.NewGetOperationalStateUseCase(
		nil,
		&stubGateReader{gate: gate},
		monitoring.SurfaceAvailability{},
	)

	reply, prob := uc.Execute(context.Background(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.State.Gate.Status != "halted" {
		t.Errorf("Gate.Status = %q, want halted", reply.State.Gate.Status)
	}
	if reply.State.Gate.Reason != "emergency stop" {
		t.Errorf("Gate.Reason = %q, want 'emergency stop'", reply.State.Gate.Reason)
	}
}
