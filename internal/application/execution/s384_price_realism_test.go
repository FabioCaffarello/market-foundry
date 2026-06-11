package execution_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/domain/instrument"
	"internal/shared/problem"
)

// ==========================================================================
// S384 — G1: Price realism via PriceSource injection
//
// Tests that DryRunSubmitter and PaperVenueAdapter use last observed market
// price from PriceSource when available, and fall back to "0" when not.
// ==========================================================================

// ---------- mock PriceSource ----------

type mockPriceSource struct {
	prices map[string]string // key: "source.symbol.timeframe" → price
	err    *problem.Problem
}

func (m *mockPriceSource) LastPrice(_ context.Context, source string, symbol instrument.CanonicalInstrument, _ int) (string, *problem.Problem) {
	if m.err != nil {
		return "0", m.err
	}
	if price, ok := m.prices[source+"."+symbol.SubjectToken()]; ok {
		return price, nil
	}
	return "0", nil
}

func newMockPriceSource(prices map[string]string) *mockPriceSource {
	return &mockPriceSource{prices: prices}
}

func newFailingPriceSource() *mockPriceSource {
	return &mockPriceSource{
		err: problem.New(problem.Unavailable, "NATS KV unavailable"),
	}
}

// ---------- DryRunSubmitter + PriceSource ----------

func TestS384_DryRun_UsesRealisticPrice(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"binancef.btc_usdt_perpetual": "67432.50",
	})
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

	intent := makeIntent(t, domainexec.SideBuy)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Intent.Fills[0].Price != "67432.50" {
		t.Errorf("expected realistic price 67432.50, got %s", receipt.Intent.Fills[0].Price)
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("fill must still be marked Simulated=true")
	}
}

func TestS384_DryRun_FallsBackToZeroWhenNoPriceSource(t *testing.T) {
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner) // no WithPriceSource

	intent := makeIntent(t, domainexec.SideBuy)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("expected fallback price 0, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_DryRun_FallsBackToZeroOnPriceError(t *testing.T) {
	ps := newFailingPriceSource()
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

	intent := makeIntent(t, domainexec.SideBuy)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("expected fallback price 0 on error, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_DryRun_FallsBackToZeroForUnknownSymbol(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"binancef.btc_usdt_perpetual": "67432.50",
	})
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

	intent := makeIntent(t, domainexec.SideBuy)
	// venue symbol "unknowncoin" — not in price source; build via "UNKNOWN" base.
	unknownInst, setupProb := instrument.New("UNKNOWN", "COIN", instrument.ContractPerpetual)
	if setupProb != nil {
		t.Fatalf("setup: %v", setupProb)
	}
	intent.Instrument = unknownInst
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("expected fallback price 0 for unknown symbol, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_DryRun_PriceDoesNotAffectNoActionIntents(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"binancef.btc_usdt_perpetual": "67432.50",
	})
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

	intent := makeIntent(t, domainexec.SideNone)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	// No-action intents produce no fills — price lookup should not be invoked.
	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("no-action intent should have 0 fills, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Status != domainexec.StatusAccepted {
		t.Errorf("expected accepted, got %s", receipt.Status)
	}
}

func TestS384_DryRun_RealisticPricePreservesOtherFields(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"binancef.btc_usdt_perpetual": "67432.50",
	})
	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

	intent := makeIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "corr-g1-test"
	intent.CausationID = "cause-g1-test"

	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}

	// Verify all structural properties are preserved.
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected filled, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
	if receipt.Intent.CorrelationID != "corr-g1-test" {
		t.Errorf("correlation_id lost: got %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "cause-g1-test" {
		t.Errorf("causation_id lost: got %s", receipt.Intent.CausationID)
	}
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Errorf("filled_quantity wrong: got %s", receipt.Intent.FilledQuantity)
	}
	if receipt.Intent.Fills[0].Simulated != true {
		t.Error("fill must remain Simulated=true")
	}
}

// ---------- PaperVenueAdapter + PriceSource ----------

func TestS384_Paper_UsesRealisticPrice(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"test.btc_usdt_perpetual": "67432.50",
	})
	adapter := appexec.NewPaperVenueAdapter(0).WithPriceSource(ps)

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

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Intent.Fills[0].Price != "67432.50" {
		t.Errorf("expected realistic price 67432.50, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_Paper_FallsBackToZeroWhenNoPriceSource(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0) // no WithPriceSource

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

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("expected fallback price 0, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_Paper_FallsBackToZeroOnPriceError(t *testing.T) {
	ps := newFailingPriceSource()
	adapter := appexec.NewPaperVenueAdapter(0).WithPriceSource(ps)

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

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("expected fallback price 0 on error, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_Paper_NoActionIntentIgnoresPriceSource(t *testing.T) {
	ps := newMockPriceSource(map[string]string{
		"test.btc_usdt_perpetual": "67432.50",
	})
	adapter := appexec.NewPaperVenueAdapter(0).WithPriceSource(ps)

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

	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("no-action intent should have 0 fills, got %d", len(receipt.Intent.Fills))
	}
}

// ---------- Backward compatibility ----------

func TestS384_BackwardCompat_DryRunWithoutPriceSourceUnchanged(t *testing.T) {
	// Existing code that doesn't call WithPriceSource must continue to work
	// identically — fills get Price="0".
	spy := &spyVenueAdapter{}
	sub := appexec.NewDryRunSubmitter(spy)

	intent := makeIntent(t, domainexec.SideBuy)
	receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}
	if spy.called {
		t.Error("inner adapter should not be called")
	}
	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("without PriceSource, price should be 0, got %s", receipt.Intent.Fills[0].Price)
	}
}

func TestS384_BackwardCompat_PaperWithoutPriceSourceUnchanged(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)

	intent := domainexec.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "test",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideSell,
		Quantity:   "1.0",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "medium",
			Timeframe:   60,
		},
		Timestamp: time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if receipt.Intent.Fills[0].Price != "0" {
		t.Errorf("without PriceSource, price should be 0, got %s", receipt.Intent.Fills[0].Price)
	}
}
