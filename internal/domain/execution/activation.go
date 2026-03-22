package execution

import "time"

// ActivationDimension represents one of the three independent dimensions that
// compose the effective venue activation state.
type ActivationDimension string

const (
	DimensionAdapter    ActivationDimension = "adapter"
	DimensionGate       ActivationDimension = "gate"
	DimensionCredential ActivationDimension = "credential"
)

// AdapterState represents the binary-level venue adapter type.
// Immutable per process lifetime — changes require binary restart.
type AdapterState string

const (
	AdapterPaper AdapterState = "paper"
	AdapterVenue AdapterState = "venue"
)

// CredentialState represents whether venue credentials are present.
// Immutable per process lifetime — changes require binary restart.
type CredentialState string

const (
	CredentialPresent CredentialState = "present"
	CredentialAbsent  CredentialState = "absent"
)

// EffectiveMode is the canonical activation outcome derived from the three dimensions.
type EffectiveMode string

const (
	// ModePaper means orders go to the paper simulator — no real venue interaction.
	ModePaper EffectiveMode = "paper"
	// ModeVenueHalted means the venue adapter is loaded but execution is blocked by the gate.
	ModeVenueHalted EffectiveMode = "venue_halted"
	// ModeVenueLive means the venue adapter is loaded, gate is active, and credentials are present.
	// This is the ONLY mode that produces real venue orders.
	ModeVenueLive EffectiveMode = "venue_live"
	// ModeVenueDegraded means the venue adapter is loaded but credentials are absent.
	// This state should not occur in production (binary should exit on missing credentials).
	ModeVenueDegraded EffectiveMode = "venue_degraded"
)

// ActivationSurface is the canonical composite of all three activation dimensions.
// It is the single source of truth for "what is this deployment doing right now?"
//
// Authority chain:
//   - AdapterState: set at binary startup from config (venue.type)
//   - GateState: runtime-mutable via HTTP PUT /execution/control → NATS KV
//   - CredentialState: set at binary startup from environment variables
//
// The EffectiveMode is derived, never stored — it is always computed from the three inputs.
type ActivationSurface struct {
	Adapter     AdapterState    `json:"adapter"`
	Gate        ControlGate     `json:"gate"`
	Credentials CredentialState `json:"credentials"`
	Effective   EffectiveMode   `json:"effective"`
	ObservedAt  time.Time       `json:"observed_at"`
}

// ComputeEffectiveMode derives the canonical activation mode from the three dimensions.
//
// Truth table:
//
//	Adapter  | Gate    | Credentials | Effective
//	---------|---------|-------------|------------------
//	paper    | *       | *           | paper
//	venue    | halted  | *           | venue_halted
//	venue    | active  | absent      | venue_degraded
//	venue    | active  | present     | venue_live
func ComputeEffectiveMode(adapter AdapterState, gate GateStatus, creds CredentialState) EffectiveMode {
	if adapter == AdapterPaper {
		return ModePaper
	}
	if gate == GateHalted {
		return ModeVenueHalted
	}
	if creds == CredentialAbsent {
		return ModeVenueDegraded
	}
	return ModeVenueLive
}

// NewActivationSurface computes the canonical activation surface from the three dimensions.
func NewActivationSurface(adapter AdapterState, gate ControlGate, creds CredentialState) ActivationSurface {
	return ActivationSurface{
		Adapter:     adapter,
		Gate:        gate,
		Credentials: creds,
		Effective:   ComputeEffectiveMode(adapter, gate.Status, creds),
		ObservedAt:  time.Now().UTC(),
	}
}

// ActivationDimensions holds the process-local activation dimensions (adapter + credentials)
// that the execute binary publishes to KV at startup. These are immutable per process lifetime.
// The store query responder reads these to compose the full ActivationSurface.
type ActivationDimensions struct {
	Adapter     AdapterState    `json:"adapter"`
	Credentials CredentialState `json:"credentials"`
	ReportedAt  time.Time       `json:"reported_at"`
	ReportedBy  string          `json:"reported_by"`
}

// IsLive reports whether this surface allows real venue execution.
func (s ActivationSurface) IsLive() bool {
	return s.Effective == ModeVenueLive
}

// CanReachVenue reports whether the adapter is configured for a real venue,
// regardless of gate or credential state.
func (s ActivationSurface) CanReachVenue() bool {
	return s.Adapter == AdapterVenue
}
