package healthz_test

import (
	"internal/shared/healthz"
	"testing"
)

func TestSegmentHealthRegistry_EmptyReturnsNoStatus(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	status := reg.Status()
	if len(status) != 0 {
		t.Fatalf("expected 0 segments, got %d", len(status))
	}
	if !reg.IsHealthy() {
		t.Fatal("empty registry should be healthy")
	}
}

func TestSegmentHealthRegistry_SingleSegmentReady(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, tracker)

	status := reg.Status()
	if len(status) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(status))
	}
	if status[0].Segment != "spot" {
		t.Errorf("expected segment 'spot', got %q", status[0].Segment)
	}
	if status[0].Phase != "ready" {
		t.Errorf("expected phase 'ready', got %q", status[0].Phase)
	}
	if !status[0].Enabled {
		t.Error("expected enabled=true")
	}
	if status[0].Adapter != "binance_spot_testnet" {
		t.Errorf("expected adapter 'binance_spot_testnet', got %q", status[0].Adapter)
	}
}

func TestSegmentHealthRegistry_DisabledSegment(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	reg.Register(healthz.SegmentDescriptor{
		Name:    "futures",
		Enabled: false,
		Adapter: "",
	}, nil)

	status := reg.Status()
	if len(status) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(status))
	}
	if status[0].Phase != "disabled" {
		t.Errorf("expected phase 'disabled', got %q", status[0].Phase)
	}
	if status[0].Enabled {
		t.Error("expected enabled=false")
	}
}

func TestSegmentHealthRegistry_ActiveAfterProcessing(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, tracker)

	// Simulate processing a spot intent.
	tracker.Counter("spot:processed").Add(5)
	tracker.Counter("spot:filled").Add(3)
	tracker.Counter("spot:rejected").Add(1)

	status := reg.Status()
	if status[0].Phase != "active" {
		t.Errorf("expected phase 'active', got %q", status[0].Phase)
	}
	if status[0].Processed != 5 {
		t.Errorf("expected processed=5, got %d", status[0].Processed)
	}
	if status[0].Filled != 3 {
		t.Errorf("expected filled=3, got %d", status[0].Filled)
	}
	if status[0].Rejected != 1 {
		t.Errorf("expected rejected=1, got %d", status[0].Rejected)
	}
}

func TestSegmentHealthRegistry_DegradedOnErrorsOnly(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	reg.Register(healthz.SegmentDescriptor{
		Name:    "futures",
		Enabled: true,
		Adapter: "binance_futures_testnet",
	}, tracker)

	// Only errors, no processed — degraded.
	tracker.Counter("futures:errors").Add(3)

	status := reg.Status()
	if status[0].Phase != "degraded" {
		t.Errorf("expected phase 'degraded', got %q", status[0].Phase)
	}
	if status[0].Errors != 3 {
		t.Errorf("expected errors=3, got %d", status[0].Errors)
	}

	if reg.IsHealthy() {
		t.Error("expected IsHealthy()=false when segment is degraded")
	}
}

func TestSegmentHealthRegistry_MultiSegmentCanonicalOrder(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	// Register spot first, then futures.
	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, tracker)
	reg.Register(healthz.SegmentDescriptor{
		Name:    "futures",
		Enabled: true,
		Adapter: "binance_futures_testnet",
	}, tracker)

	status := reg.Status()
	if len(status) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(status))
	}
	// Canonical order: futures first, then spot (alphabetical).
	if status[0].Segment != "futures" {
		t.Errorf("expected first segment 'futures', got %q", status[0].Segment)
	}
	if status[1].Segment != "spot" {
		t.Errorf("expected second segment 'spot', got %q", status[1].Segment)
	}
}

func TestSegmentHealthRegistry_MultiSegmentIndependentPhases(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, tracker)
	reg.Register(healthz.SegmentDescriptor{
		Name:    "futures",
		Enabled: true,
		Adapter: "binance_futures_testnet",
	}, tracker)

	// Spot is active, futures has only errors.
	tracker.Counter("spot:processed").Add(10)
	tracker.Counter("futures:errors").Add(2)

	status := reg.Status()
	// futures first in canonical order.
	if status[0].Phase != "degraded" {
		t.Errorf("expected futures phase 'degraded', got %q", status[0].Phase)
	}
	if status[1].Phase != "active" {
		t.Errorf("expected spot phase 'active', got %q", status[1].Phase)
	}

	if reg.IsHealthy() {
		t.Error("expected IsHealthy()=false — futures is degraded")
	}
}

func TestSegmentHealthRegistry_SegmentPhase(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	tracker := healthz.NewTracker("venue-adapter")

	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, tracker)

	if phase := reg.SegmentPhase("spot"); phase != "ready" {
		t.Errorf("expected phase 'ready', got %q", phase)
	}
	if phase := reg.SegmentPhase("futures"); phase != "unknown" {
		t.Errorf("expected phase 'unknown' for unregistered segment, got %q", phase)
	}
}

func TestSegmentHealthRegistry_NilTrackerReady(t *testing.T) {
	reg := healthz.NewSegmentHealthRegistry()
	reg.Register(healthz.SegmentDescriptor{
		Name:    "spot",
		Enabled: true,
		Adapter: "binance_spot_testnet",
	}, nil)

	status := reg.Status()
	if status[0].Phase != "ready" {
		t.Errorf("expected phase 'ready' with nil tracker, got %q", status[0].Phase)
	}
	if status[0].Processed != 0 {
		t.Errorf("expected processed=0 with nil tracker, got %d", status[0].Processed)
	}
}
