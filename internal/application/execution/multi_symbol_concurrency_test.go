package execution_test

// S304: Execution Paper Behavior Under Multi-Symbol Concurrency
//
// Validates that paper order evaluation, fill simulation, and venue adaptation
// produce correct, isolated, and explainable results when multiple symbols
// are evaluated simultaneously with different risk dispositions.
//
// Scenarios:
//   EX-1 — Three symbols with different dispositions → correct side/quantity per symbol.
//   EX-2 — Full paper lifecycle per symbol (evaluate → simulate fill).
//   EX-3 — Rejected risk blocks execution per symbol, no cross-symbol leakage.
//   EX-4 — Modified disposition preserves risk-adjusted quantity per symbol.
//   EX-5 — Causal context (strategy type, severity) preserved through risk→exec boundary.
//   EX-6 — Paper venue adapter isolation per symbol.

import (
	"context"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

// ---------------------------------------------------------------------------
// EX-1: Three symbols, different dispositions → correct side/quantity
// ---------------------------------------------------------------------------

func TestS304_EX1_MultiSymbolDispositionMapping(t *testing.T) {
	ts := time.Now()

	type symbolCase struct {
		symbol    string
		riskDisp  string
		direction string
		maxPosPct string
		wantSide  domainexec.Side
		wantQty   string
	}

	cases := []symbolCase{
		{symbol: "btcusdt", riskDisp: "approved", direction: "long", maxPosPct: "0.0200", wantSide: domainexec.SideBuy, wantQty: "0.0200"},
		{symbol: "ethusdt", riskDisp: "rejected", direction: "long", maxPosPct: "0.0200", wantSide: domainexec.SideNone, wantQty: "0"},
		{symbol: "solusdt", riskDisp: "approved", direction: "short", maxPosPct: "0.0500", wantSide: domainexec.SideSell, wantQty: "0.0500"},
	}

	for _, sc := range cases {
		t.Run(sc.symbol, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sc.symbol), 60)
			intent, ok := eval.Evaluate(
				"position_exposure", sc.riskDisp, "0.85", sc.maxPosPct,
				sc.direction, "0.72",
				"mean_reversion_entry", "high",
				60, ts,
			)
			if !ok {
				t.Fatal("evaluation failed")
			}
			if intent.Side != sc.wantSide {
				t.Errorf("side=%q, want %q", intent.Side, sc.wantSide)
			}
			if intent.Quantity != sc.wantQty {
				t.Errorf("quantity=%q, want %q", intent.Quantity, sc.wantQty)
			}
			if intent.VenueSymbol() != sc.symbol {
				t.Errorf("symbol bleed: got %q", intent.VenueSymbol())
			}
			if prob := intent.Validate(); prob != nil {
				t.Errorf("validation failed: %s", prob.Message)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EX-2: Full paper lifecycle per symbol (evaluate → simulate fill)
// ---------------------------------------------------------------------------

func TestS304_EX2_FullPaperLifecyclePerSymbol(t *testing.T) {
	ts := time.Now()
	sim := &appexec.PaperFillSimulator{}

	type symbolCase struct {
		symbol    string
		direction string
		wantSide  domainexec.Side
	}

	cases := []symbolCase{
		{symbol: "btcusdt", direction: "long", wantSide: domainexec.SideBuy},
		{symbol: "ethusdt", direction: "short", wantSide: domainexec.SideSell},
		{symbol: "solusdt", direction: "long", wantSide: domainexec.SideBuy},
	}

	for _, sc := range cases {
		t.Run(sc.symbol, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sc.symbol), 60)
			intent, ok := eval.Evaluate(
				"position_exposure", "approved", "0.85", "0.0200",
				sc.direction, "0.72",
				"mean_reversion_entry", "moderate",
				60, ts,
			)
			if !ok {
				t.Fatal("evaluation failed")
			}
			if intent.Status != domainexec.StatusSubmitted {
				t.Fatalf("pre-fill status=%q, want submitted", intent.Status)
			}

			// Simulate fill.
			filled, ok := sim.SimulateFill(intent)
			if !ok {
				t.Fatal("fill simulation failed")
			}

			// Post-fill assertions.
			if filled.Status != domainexec.StatusFilled {
				t.Errorf("post-fill status=%q, want filled", filled.Status)
			}
			if filled.FilledQuantity != intent.Quantity {
				t.Errorf("filled_quantity=%q, want %q", filled.FilledQuantity, intent.Quantity)
			}
			if len(filled.Fills) != 1 {
				t.Fatalf("expected 1 fill record, got %d", len(filled.Fills))
			}
			if !filled.Fills[0].Simulated {
				t.Error("fill record should be simulated=true")
			}

			// Symbol survives lifecycle.
			if filled.VenueSymbol() != sc.symbol {
				t.Errorf("symbol bleed after fill: got %q", filled.VenueSymbol())
			}
			if filled.Side != sc.wantSide {
				t.Errorf("side changed after fill: got %q", filled.Side)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EX-3: Rejected risk blocks execution per symbol
// ---------------------------------------------------------------------------

func TestS304_EX3_RejectedBlocksExecution(t *testing.T) {
	ts := time.Now()
	sim := &appexec.PaperFillSimulator{}

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	for _, sym := range symbols {
		t.Run(sym, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sym), 60)
			intent, ok := eval.Evaluate(
				"position_exposure", "rejected", "0.30", "0.0200",
				"long", "0.72",
				"mean_reversion_entry", "low",
				60, ts,
			)
			if !ok {
				t.Fatal("evaluation failed")
			}

			// Rejected → side=none, quantity=0.
			if intent.Side != domainexec.SideNone {
				t.Errorf("side=%q, want none", intent.Side)
			}
			if intent.Quantity != "0" {
				t.Errorf("quantity=%q, want 0", intent.Quantity)
			}

			// Fill simulation: no-action intent returns unchanged.
			filled, ok := sim.SimulateFill(intent)
			if !ok {
				t.Fatal("fill simulation should succeed for no-action")
			}
			if filled.Status != domainexec.StatusSubmitted {
				t.Errorf("rejected intent should remain submitted, got %q", filled.Status)
			}
			if len(filled.Fills) != 0 {
				t.Errorf("rejected intent should have no fills, got %d", len(filled.Fills))
			}

			// Symbol isolation.
			if filled.VenueSymbol() != sym {
				t.Errorf("symbol bleed: got %q", filled.VenueSymbol())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EX-4: Modified disposition preserves risk-adjusted quantity
// ---------------------------------------------------------------------------

func TestS304_EX4_ModifiedQuantityPerSymbol(t *testing.T) {
	ts := time.Now()
	sim := &appexec.PaperFillSimulator{}

	type symbolCase struct {
		symbol    string
		maxPos    string
		direction string
		wantSide  domainexec.Side
	}

	cases := []symbolCase{
		{symbol: "btcusdt", maxPos: "0.0100", direction: "long", wantSide: domainexec.SideBuy},
		{symbol: "ethusdt", maxPos: "0.0080", direction: "short", wantSide: domainexec.SideSell},
		{symbol: "solusdt", maxPos: "0.0050", direction: "long", wantSide: domainexec.SideBuy},
	}

	for _, sc := range cases {
		t.Run(sc.symbol, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sc.symbol), 60)
			intent, ok := eval.Evaluate(
				"position_exposure", "modified", "0.60", sc.maxPos,
				sc.direction, "0.72",
				"mean_reversion_entry", "moderate",
				60, ts,
			)
			if !ok {
				t.Fatal("evaluation failed")
			}

			if intent.Side != sc.wantSide {
				t.Errorf("side=%q, want %q", intent.Side, sc.wantSide)
			}
			if intent.Quantity != sc.maxPos {
				t.Errorf("quantity=%q, want %q (risk-adjusted)", intent.Quantity, sc.maxPos)
			}

			// Fill simulation applies to modified.
			filled, ok := sim.SimulateFill(intent)
			if !ok {
				t.Fatal("fill simulation failed")
			}
			if filled.Status != domainexec.StatusFilled {
				t.Errorf("post-fill status=%q, want filled", filled.Status)
			}
			if filled.FilledQuantity != sc.maxPos {
				t.Errorf("filled_quantity=%q, want %q", filled.FilledQuantity, sc.maxPos)
			}
			if filled.VenueSymbol() != sc.symbol {
				t.Errorf("symbol bleed after fill: got %q", filled.VenueSymbol())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EX-5: Causal context preserved through risk→exec boundary
// ---------------------------------------------------------------------------

func TestS304_EX5_CausalContextPreservation(t *testing.T) {
	ts := time.Now()

	type symbolCase struct {
		symbol       string
		strategyType string
		severity     string
		riskType     string
	}

	cases := []symbolCase{
		{symbol: "btcusdt", strategyType: "mean_reversion_entry", severity: "high", riskType: "position_exposure"},
		{symbol: "ethusdt", strategyType: "trend_following_entry", severity: "moderate", riskType: "drawdown_limit"},
		{symbol: "solusdt", strategyType: "squeeze_breakout_entry", severity: "low", riskType: "position_exposure"},
	}

	for _, sc := range cases {
		t.Run(sc.symbol, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sc.symbol), 60)
			intent, ok := eval.Evaluate(
				sc.riskType, "approved", "0.85", "0.0200",
				"long", "0.72",
				sc.strategyType, sc.severity,
				60, ts,
			)
			if !ok {
				t.Fatal("evaluation failed")
			}

			// RiskInput preserves strategy type and severity.
			if intent.Risk.StrategyType != sc.strategyType {
				t.Errorf("risk.strategy_type=%q, want %q", intent.Risk.StrategyType, sc.strategyType)
			}
			if intent.Risk.DecisionSeverity != sc.severity {
				t.Errorf("risk.decision_severity=%q, want %q", intent.Risk.DecisionSeverity, sc.severity)
			}
			if intent.Risk.Type != sc.riskType {
				t.Errorf("risk.type=%q, want %q", intent.Risk.Type, sc.riskType)
			}

			// Parameters carry context.
			if intent.Parameters["strategy_type"] != sc.strategyType {
				t.Errorf("params.strategy_type=%q", intent.Parameters["strategy_type"])
			}
			if intent.Parameters["decision_severity"] != sc.severity {
				t.Errorf("params.decision_severity=%q", intent.Parameters["decision_severity"])
			}

			// Symbol isolation.
			if intent.VenueSymbol() != sc.symbol {
				t.Errorf("symbol bleed: got %q", intent.VenueSymbol())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EX-6: Paper venue adapter isolation per symbol
// ---------------------------------------------------------------------------

func TestS304_EX6_PaperVenueAdapterIsolation(t *testing.T) {
	ts := time.Now()
	adapter := appexec.NewPaperVenueAdapter(0)

	type symbolCase struct {
		symbol    string
		direction string
		wantSide  domainexec.Side
	}

	cases := []symbolCase{
		{symbol: "btcusdt", direction: "long", wantSide: domainexec.SideBuy},
		{symbol: "ethusdt", direction: "short", wantSide: domainexec.SideSell},
		{symbol: "solusdt", direction: "long", wantSide: domainexec.SideBuy},
	}

	venueOrderIDs := map[string]string{}

	for _, sc := range cases {
		t.Run(sc.symbol, func(t *testing.T) {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sc.symbol), 60)
			intent, _ := eval.Evaluate(
				"position_exposure", "approved", "0.85", "0.0200",
				sc.direction, "0.72",
				"mean_reversion_entry", "moderate",
				60, ts,
			)

			// Submit to venue adapter with the intent.
			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("venue submit failed: %s", prob.Message)
			}

			// Receipt must have a venue order ID.
			if receipt.VenueOrderID == "" {
				t.Error("missing venue order ID")
			}

			// Venue order IDs must be unique per symbol.
			if existing, found := venueOrderIDs[receipt.VenueOrderID]; found {
				t.Fatalf("venue order ID collision between %s and %s", existing, sc.symbol)
			}
			venueOrderIDs[receipt.VenueOrderID] = sc.symbol

			// Intent inside receipt carries the correct symbol.
			if receipt.Intent.VenueSymbol() != sc.symbol {
				t.Errorf("receipt.intent.symbol=%q, want %q", receipt.Intent.VenueSymbol(), sc.symbol)
			}

			// Receipt status is filled (paper venue fills instantly).
			if receipt.Status != domainexec.StatusFilled {
				t.Errorf("receipt.status=%q, want filled", receipt.Status)
			}
		})
	}
}
