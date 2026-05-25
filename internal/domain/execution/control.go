package execution

import (
	"time"

	"internal/shared/clock"
)

// GateStatus represents the operational state of the execution control gate.
type GateStatus string

const (
	GateActive GateStatus = "active"
	GateHalted GateStatus = "halted"
)

// ValidGateStatus reports whether s is a recognized gate status value.
func ValidGateStatus(s GateStatus) bool {
	return s == GateActive || s == GateHalted
}

// ControlGate represents the operational control state for execution pipelines.
// Authority: store binary (KV bucket). Readers: derive publisher, gateway.
// Key: "global" — single gate for all execution families in this deployment.
type ControlGate struct {
	Status    GateStatus `json:"status"`
	Reason    string     `json:"reason,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	UpdatedBy string     `json:"updated_by,omitempty"`
}

// IsHalted reports whether the gate blocks execution publishing.
func (g ControlGate) IsHalted() bool {
	return g.Status == GateHalted
}

// DefaultControlGate returns an active gate with no restrictions.
// Receives time via clock.Clock per ADR-0019 INV-D1. The caller
// is responsible for passing a non-nil Clock; nil-handling for
// the natsexecution.ControlKVStore.Get fail-open path lives in
// the adapter (separating nil-receiver from nil-bucket cases).
func DefaultControlGate(clk clock.Clock) ControlGate {
	return ControlGate{
		Status:    GateActive,
		UpdatedAt: clk.Now().UTC(),
	}
}
