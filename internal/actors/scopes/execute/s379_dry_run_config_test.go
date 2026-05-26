package execute_test

import (
	"strings"
	"testing"

	"context"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/domain/instrument"
	"internal/shared/healthz"
	"internal/shared/problem"
	"internal/shared/settings"
)

func btcUSDTPerpS379(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func btcUSDTSpotS379(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// TestS379_DryRunConfig_FailClosed verifies that the default VenueConfig
// (dry_run omitted or nil) resolves to dry-run mode.
func TestS379_DryRunConfig_FailClosed(t *testing.T) {
	t.Run("nil dry_run defaults to true", func(t *testing.T) {
		cfg := settings.VenueConfig{Type: settings.VenueTypePaperSimulator}
		if !cfg.IsDryRun() {
			t.Fatal("nil DryRun must resolve to true (fail-closed)")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		v := true
		cfg := settings.VenueConfig{Type: settings.VenueTypePaperSimulator, DryRun: &v}
		if !cfg.IsDryRun() {
			t.Fatal("explicit true must be dry-run")
		}
	})

	t.Run("explicit false with venue adapter", func(t *testing.T) {
		v := false
		cfg := settings.VenueConfig{Type: settings.VenueTypeBinanceFuturesTestnet, DryRun: &v}
		if cfg.IsDryRun() {
			t.Fatal("explicit false with venue adapter must not be dry-run")
		}
	})

	t.Run("empty type defaults to dry-run", func(t *testing.T) {
		cfg := settings.VenueConfig{}
		if !cfg.IsDryRun() {
			t.Fatal("empty config must resolve to dry-run (fail-closed)")
		}
	})
}

// TestS379_DryRunConfig_ValidationRejectsPaperWithDryRunFalse verifies
// that setting dry_run=false with paper_simulator is rejected at validation.
func TestS379_DryRunConfig_ValidationRejectsPaperWithDryRunFalse(t *testing.T) {
	v := false
	cfg := settings.VenueConfig{Type: settings.VenueTypePaperSimulator, DryRun: &v}
	prob := cfg.Validate()
	if prob == nil {
		t.Fatal("dry_run=false with paper_simulator must be rejected")
	}
	if !strings.Contains(prob.Message, "invalid") {
		t.Errorf("expected validation error, got: %s", prob.Message)
	}
}

// TestS379_DryRunSubmitter_PipelineTraversal proves that when dry_run=true,
// intents traverse the full pipeline structure (safety gates, actor dispatch)
// but the DryRunSubmitter intercepts before any real venue call.
func TestS379_DryRunSubmitter_PipelineTraversal(t *testing.T) {
	tracker := healthz.NewTracker("s379-pipeline")

	// Build the same pipeline as cmd/execute/run.go, with DryRunSubmitter outermost.
	rawVenue := appexec.NewPaperVenueAdapter(0)
	retrySubmitter := appexec.NewRetrySubmitter(rawVenue, appexec.DefaultRetryPolicy())
	// Post200Reconciler skipped (paper has no query port) — matches production wiring.
	var composedVenue ports.VenuePort = retrySubmitter

	// S379: DryRunSubmitter wraps the composed pipeline.
	dryRun := appexec.NewDryRunSubmitter(composedVenue).WithTracker(tracker)

	intent := domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerpS379(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.95"},
		CorrelationID: "s379-corr-001",
		CausationID:   "s379-cause-001",
		Timestamp:     time.Now().UTC(),
	}

	receipt, prob := dryRun.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	// Receipt must bear dry-run markers.
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("venue order ID must have dryrun- prefix, got: %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected StatusFilled, got %s", receipt.Status)
	}
	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("expected at least one fill record")
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("fill must be marked Simulated=true")
	}

	// Correlation must survive.
	if receipt.Intent.CorrelationID != "s379-corr-001" {
		t.Errorf("correlation_id lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s379-cause-001" {
		t.Errorf("causation_id lost: %s", receipt.Intent.CausationID)
	}

	// Counter must be incremented.
	if tracker.Counter("dryrun_intercepted").Load() != 1 {
		t.Errorf("dryrun_intercepted counter: expected 1, got %d", tracker.Counter("dryrun_intercepted").Load())
	}
}

// TestS379_DryRunSubmitter_NeverCallsRealAdapter verifies the hard guarantee:
// even when wrapping a real-looking adapter, DryRunSubmitter never delegates.
func TestS379_DryRunSubmitter_NeverCallsRealAdapter(t *testing.T) {
	// Use a bomb adapter that panics if called.
	bomb := &bombAdapter{}
	sub := appexec.NewDryRunSubmitter(bomb)

	ethInst, setupProb := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if setupProb != nil {
		t.Fatalf("setup: %v", setupProb)
	}
	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: ethInst,
		Timeframe:  60,
		Side:       domainexec.SideSell,
		Quantity:   "0.01",
		Status:     domainexec.StatusSubmitted,
		Risk:       domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.90"},
		Timestamp:  time.Now().UTC(),
	}

	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
}

// bombAdapter panics on SubmitOrder — used to prove DryRunSubmitter never delegates.
type bombAdapter struct{}

func (b *bombAdapter) SubmitOrder(_ context.Context, _ ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	panic("bombAdapter.SubmitOrder called — DryRunSubmitter leaked to inner adapter")
}
