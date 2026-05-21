package execution_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// spyVenueAdapter records whether SubmitOrder was called.
type spyVenueAdapter struct {
	called bool
}

func (s *spyVenueAdapter) SubmitOrder(_ context.Context, _ ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	s.called = true
	return ports.VenueOrderReceipt{}, nil
}

func makeIntent(side domainexec.Side) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Symbol:        "btcusdt",
		Timeframe:     60,
		Side:          side,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.95"},
		CorrelationID: "test-corr-001",
		CausationID:   "test-cause-001",
		Timestamp:     time.Now().UTC(),
	}
}

func TestDryRunSubmitter_InterceptsBuyIntent(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("test-dryrun")
	sub := appexec.NewDryRunSubmitter(inner).WithTracker(tracker)

	intent := makeIntent(domainexec.SideBuy)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected StatusFilled, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("expected fill to be marked Simulated=true")
	}
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Errorf("expected filled_quantity=0.001, got %s", receipt.Intent.FilledQuantity)
	}

	// Verify counters.
	if tracker.Counter("dryrun_intercepted").Load() != 1 {
		t.Errorf("expected dryrun_intercepted=1, got %d", tracker.Counter("dryrun_intercepted").Load())
	}
	if tracker.Counter("dryrun_filled").Load() != 1 {
		t.Errorf("expected dryrun_filled=1, got %d", tracker.Counter("dryrun_filled").Load())
	}
}

func TestDryRunSubmitter_InterceptsSellIntent(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner)

	intent := makeIntent(domainexec.SideSell)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected StatusFilled, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
	if receipt.Intent.Side != domainexec.SideSell {
		t.Errorf("expected side=sell, got %s", receipt.Intent.Side)
	}
}

func TestDryRunSubmitter_InterceptsNoActionIntent(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("test-dryrun-noop")
	sub := appexec.NewDryRunSubmitter(inner).WithTracker(tracker)

	intent := makeIntent(domainexec.SideNone)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusAccepted {
		t.Errorf("expected StatusAccepted for SideNone, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("expected 0 fills for SideNone, got %d", len(receipt.Intent.Fills))
	}

	if tracker.Counter("dryrun_noop").Load() != 1 {
		t.Errorf("expected dryrun_noop=1, got %d", tracker.Counter("dryrun_noop").Load())
	}
}

func TestDryRunSubmitter_NeverDelegatesToInner(t *testing.T) {
	// Verify that the inner adapter is never called, even if it exists.
	// We use a PaperVenueAdapter as inner — if DryRunSubmitter incorrectly
	// delegates, the receipt would have a "paper-" prefix instead of "dryrun-".
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner)

	for _, side := range []domainexec.Side{domainexec.SideBuy, domainexec.SideSell, domainexec.SideNone} {
		intent := makeIntent(side)
		receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
		if prob != nil {
			t.Fatalf("side=%s: unexpected problem: %s", side, prob.Message)
		}
		if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
			t.Errorf("side=%s: expected dryrun- prefix (inner leaked), got %s", side, receipt.VenueOrderID)
		}
	}
}

func TestDryRunSubmitter_PreservesCorrelationFields(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner)

	intent := makeIntent(domainexec.SideBuy)
	intent.CorrelationID = "corr-abc-123"
	intent.CausationID = "cause-xyz-789"

	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Intent.CorrelationID != "corr-abc-123" {
		t.Errorf("correlation_id lost: got %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "cause-xyz-789" {
		t.Errorf("causation_id lost: got %s", receipt.Intent.CausationID)
	}
}

func TestDryRunSubmitter_UniqueOrderIDs(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner)

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		intent := makeIntent(domainexec.SideBuy)
		receipt, _ := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
		if seen[receipt.VenueOrderID] {
			t.Fatalf("duplicate order ID at iteration %d: %s", i, receipt.VenueOrderID)
		}
		seen[receipt.VenueOrderID] = true
	}
}
