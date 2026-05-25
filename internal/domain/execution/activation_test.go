package execution_test

import (
	"testing"
	"time"

	"internal/domain/execution"
	"internal/shared/clock"
)

// ---------- ComputeEffectiveMode: Truth Table ----------

func TestComputeEffectiveMode_PaperAlwaysPaper(t *testing.T) {
	cases := []struct {
		gate  execution.GateStatus
		creds execution.CredentialState
	}{
		{execution.GateActive, execution.CredentialPresent},
		{execution.GateActive, execution.CredentialAbsent},
		{execution.GateHalted, execution.CredentialPresent},
		{execution.GateHalted, execution.CredentialAbsent},
	}
	for _, tc := range cases {
		mode := execution.ComputeEffectiveMode(execution.AdapterPaper, tc.gate, tc.creds)
		if mode != execution.ModePaper {
			t.Errorf("paper adapter with gate=%s creds=%s: want paper, got %s", tc.gate, tc.creds, mode)
		}
	}
}

func TestComputeEffectiveMode_VenueHalted(t *testing.T) {
	// Venue + halted = venue_halted regardless of credentials.
	for _, creds := range []execution.CredentialState{execution.CredentialPresent, execution.CredentialAbsent} {
		mode := execution.ComputeEffectiveMode(execution.AdapterVenue, execution.GateHalted, creds)
		if mode != execution.ModeVenueHalted {
			t.Errorf("venue + halted + creds=%s: want venue_halted, got %s", creds, mode)
		}
	}
}

func TestComputeEffectiveMode_VenueActivePresentIsLive(t *testing.T) {
	mode := execution.ComputeEffectiveMode(execution.AdapterVenue, execution.GateActive, execution.CredentialPresent)
	if mode != execution.ModeVenueLive {
		t.Fatalf("venue + active + present: want venue_live, got %s", mode)
	}
}

func TestComputeEffectiveMode_VenueActiveAbsentIsDegraded(t *testing.T) {
	mode := execution.ComputeEffectiveMode(execution.AdapterVenue, execution.GateActive, execution.CredentialAbsent)
	if mode != execution.ModeVenueDegraded {
		t.Fatalf("venue + active + absent: want venue_degraded, got %s", mode)
	}
}

// ---------- ActivationSurface ----------

func TestNewActivationSurface_ComputesEffective(t *testing.T) {
	gate := execution.ControlGate{Status: execution.GateActive, UpdatedAt: time.Now().UTC()}

	surface := execution.NewActivationSurface(clock.SystemClock{}, execution.AdapterVenue, gate, execution.CredentialPresent)
	if surface.Effective != execution.ModeVenueLive {
		t.Fatalf("want venue_live, got %s", surface.Effective)
	}
	if !surface.IsLive() {
		t.Fatal("expected IsLive=true for venue_live")
	}
	if !surface.CanReachVenue() {
		t.Fatal("expected CanReachVenue=true for venue adapter")
	}
}

func TestNewActivationSurface_PaperNotLive(t *testing.T) {
	gate := execution.ControlGate{Status: execution.GateActive, UpdatedAt: time.Now().UTC()}

	surface := execution.NewActivationSurface(clock.SystemClock{}, execution.AdapterPaper, gate, execution.CredentialAbsent)
	if surface.IsLive() {
		t.Fatal("expected IsLive=false for paper")
	}
	if surface.CanReachVenue() {
		t.Fatal("expected CanReachVenue=false for paper adapter")
	}
}

func TestNewActivationSurface_ObservedAtIsSet(t *testing.T) {
	before := time.Now().UTC()
	gate := execution.ControlGate{Status: execution.GateActive, UpdatedAt: time.Now().UTC()}
	surface := execution.NewActivationSurface(clock.SystemClock{}, execution.AdapterPaper, gate, execution.CredentialAbsent)
	after := time.Now().UTC()

	if surface.ObservedAt.Before(before) || surface.ObservedAt.After(after) {
		t.Fatalf("ObservedAt %v not in [%v, %v]", surface.ObservedAt, before, after)
	}
}

// ---------- Exhaustive Truth Table ----------

func TestComputeEffectiveMode_ExhaustiveTruthTable(t *testing.T) {
	type row struct {
		adapter  execution.AdapterState
		gate     execution.GateStatus
		creds    execution.CredentialState
		expected execution.EffectiveMode
	}

	table := []row{
		{execution.AdapterPaper, execution.GateActive, execution.CredentialPresent, execution.ModePaper},
		{execution.AdapterPaper, execution.GateActive, execution.CredentialAbsent, execution.ModePaper},
		{execution.AdapterPaper, execution.GateHalted, execution.CredentialPresent, execution.ModePaper},
		{execution.AdapterPaper, execution.GateHalted, execution.CredentialAbsent, execution.ModePaper},
		{execution.AdapterVenue, execution.GateActive, execution.CredentialPresent, execution.ModeVenueLive},
		{execution.AdapterVenue, execution.GateActive, execution.CredentialAbsent, execution.ModeVenueDegraded},
		{execution.AdapterVenue, execution.GateHalted, execution.CredentialPresent, execution.ModeVenueHalted},
		{execution.AdapterVenue, execution.GateHalted, execution.CredentialAbsent, execution.ModeVenueHalted},
	}

	for _, r := range table {
		mode := execution.ComputeEffectiveMode(r.adapter, r.gate, r.creds)
		if mode != r.expected {
			t.Errorf("adapter=%s gate=%s creds=%s: want %s, got %s",
				r.adapter, r.gate, r.creds, r.expected, mode)
		}
	}
}
