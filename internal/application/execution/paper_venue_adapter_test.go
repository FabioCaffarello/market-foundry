package execution_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

func TestPaperVenueAdapter_SubmitOrder_Buy(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.05",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "high",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected filled, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "paper-") {
		t.Errorf("expected paper- prefix, got %s", receipt.VenueOrderID)
	}
	if receipt.Intent.FilledQuantity != "0.05" {
		t.Errorf("expected filled quantity 0.05, got %s", receipt.Intent.FilledQuantity)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("expected simulated fill")
	}
}

func TestPaperVenueAdapter_SubmitOrder_Sell(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: ethUSDTPerp(t),
		Timeframe:  300,
		Side:       domainexec.SideSell,
		Quantity:   "1.0",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "medium",
			Timeframe:   300,
		},
		Timestamp: time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.FilledQuantity != "1.0" {
		t.Errorf("expected filled quantity 1.0, got %s", receipt.Intent.FilledQuantity)
	}
}

func TestPaperVenueAdapter_SubmitOrder_NoAction(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideNone,
		Quantity:   "0",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "rejected",
			Confidence:  "low",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if receipt.Status != domainexec.StatusAccepted {
		t.Errorf("expected accepted for no-action, got %s", receipt.Status)
	}
}

func TestPaperVenueAdapter_UniqueVenueOrderIDs(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.01",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "high",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if ids[receipt.VenueOrderID] {
			t.Fatalf("duplicate venue order ID: %s", receipt.VenueOrderID)
		}
		ids[receipt.VenueOrderID] = true
	}
}

func TestPaperVenueAdapter_ImplementsVenuePort(t *testing.T) {
	var _ ports.VenuePort = (*appexec.PaperVenueAdapter)(nil)
}

func TestPaperVenueAdapter_SubmitOrder_CancelledContext(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.01",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "high",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	// Paper adapter ignores context (instant fill), so cancelled context still succeeds.
	// This documents the paper adapter's behavior — real adapters must respect context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	receipt, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("paper adapter should succeed even with cancelled context: %v", prob)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
}

func TestPaperVenueAdapter_FillDelay_RespectsDelay(t *testing.T) {
	delay := 50 * time.Millisecond
	adapter := appexec.NewPaperVenueAdapter(delay)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.01",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "high",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	start := time.Now()
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	elapsed := time.Since(start)

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if elapsed < delay {
		t.Fatalf("expected at least %v delay, got %v", delay, elapsed)
	}
}
