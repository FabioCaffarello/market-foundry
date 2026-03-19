package execution_test

import (
	"testing"
	"time"

	"internal/domain/execution"
)

// ---------- GateStatus Validation ----------

func TestValidGateStatus_Active(t *testing.T) {
	if !execution.ValidGateStatus(execution.GateActive) {
		t.Fatal("GateActive should be valid")
	}
}

func TestValidGateStatus_Halted(t *testing.T) {
	if !execution.ValidGateStatus(execution.GateHalted) {
		t.Fatal("GateHalted should be valid")
	}
}

func TestValidGateStatus_Unknown(t *testing.T) {
	if execution.ValidGateStatus("unknown") {
		t.Fatal("unknown gate status should be invalid")
	}
}

func TestValidGateStatus_Empty(t *testing.T) {
	if execution.ValidGateStatus("") {
		t.Fatal("empty gate status should be invalid")
	}
}

// ---------- IsHalted ----------

func TestControlGate_IsHalted_WhenHalted(t *testing.T) {
	gate := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "manual intervention",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "operator",
	}
	if !gate.IsHalted() {
		t.Fatal("expected IsHalted=true for halted gate")
	}
}

func TestControlGate_IsHalted_WhenActive(t *testing.T) {
	gate := execution.ControlGate{
		Status:    execution.GateActive,
		UpdatedAt: time.Now().UTC(),
	}
	if gate.IsHalted() {
		t.Fatal("expected IsHalted=false for active gate")
	}
}

func TestControlGate_IsHalted_ZeroValue(t *testing.T) {
	gate := execution.ControlGate{}
	// Zero value has empty status — should not be halted (fail-open).
	if gate.IsHalted() {
		t.Fatal("zero-value gate should not be halted (fail-open semantics)")
	}
}

// ---------- DefaultControlGate ----------

func TestDefaultControlGate_IsActive(t *testing.T) {
	gate := execution.DefaultControlGate()
	if gate.Status != execution.GateActive {
		t.Fatalf("expected GateActive, got %q", gate.Status)
	}
	if gate.IsHalted() {
		t.Fatal("default gate should not be halted")
	}
}

func TestDefaultControlGate_HasTimestamp(t *testing.T) {
	before := time.Now().UTC()
	gate := execution.DefaultControlGate()
	after := time.Now().UTC()

	if gate.UpdatedAt.Before(before) || gate.UpdatedAt.After(after) {
		t.Fatalf("expected UpdatedAt in [%v, %v], got %v", before, after, gate.UpdatedAt)
	}
}

func TestDefaultControlGate_NoReason(t *testing.T) {
	gate := execution.DefaultControlGate()
	if gate.Reason != "" {
		t.Fatalf("expected empty reason, got %q", gate.Reason)
	}
}

func TestDefaultControlGate_NoUpdatedBy(t *testing.T) {
	gate := execution.DefaultControlGate()
	if gate.UpdatedBy != "" {
		t.Fatalf("expected empty updated_by, got %q", gate.UpdatedBy)
	}
}

// ---------- Gate Transition Semantics ----------

func TestControlGate_HaltAndResume(t *testing.T) {
	// Simulate halt → resume cycle.
	gate := execution.DefaultControlGate()
	if gate.IsHalted() {
		t.Fatal("initial gate should be active")
	}

	// Halt.
	gate.Status = execution.GateHalted
	gate.Reason = "risk incident"
	gate.UpdatedBy = "oncall"
	gate.UpdatedAt = time.Now().UTC()
	if !gate.IsHalted() {
		t.Fatal("gate should be halted after setting status")
	}

	// Resume.
	gate.Status = execution.GateActive
	gate.Reason = ""
	gate.UpdatedBy = "oncall"
	gate.UpdatedAt = time.Now().UTC()
	if gate.IsHalted() {
		t.Fatal("gate should be active after resume")
	}
}

func TestControlGate_HaltPreservesAuditFields(t *testing.T) {
	gate := execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "emergency stop",
		UpdatedAt: time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC),
		UpdatedBy: "admin",
	}
	if gate.Reason != "emergency stop" {
		t.Fatalf("expected reason preserved, got %q", gate.Reason)
	}
	if gate.UpdatedBy != "admin" {
		t.Fatalf("expected updated_by preserved, got %q", gate.UpdatedBy)
	}
}
