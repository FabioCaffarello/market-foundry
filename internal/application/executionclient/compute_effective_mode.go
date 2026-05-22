package executionclient

import (
	"internal/domain/execution"
)

// ComputeEffectiveMode resolves the canonical activation mode from the three
// dimensions. It is a thin pure wrapper around execution.ComputeEffectiveMode.
//
// ADR-0005 permits cmd/ composition roots to reference domain types for
// wiring, but invoking domain functions crosses the boundary. Exposing this
// derivation at the application layer lets cmd/ call it without reaching into
// internal/domain/ for behaviour.
func ComputeEffectiveMode(
	adapter execution.AdapterState,
	gate execution.GateStatus,
	creds execution.CredentialState,
) execution.EffectiveMode {
	return execution.ComputeEffectiveMode(adapter, gate, creds)
}
