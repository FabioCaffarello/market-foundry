package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// --- Pairing Read Model Tests (S481) ---

// filledChainWithSide creates a chain with fills and the given side.
func filledChainWithSide(corrID string, side execution.Side, price, qty, fee, costBasis string, ts time.Time) *analyticalclient.CompositeExecutionChain {
	chain := fullChain(corrID)
	chain.Execution.Side = side
	chain.Execution.Status = "filled"
	chain.Execution.FilledQuantity = qty
	chain.Execution.CorrelationID = corrID
	chain.Execution.Timestamp = ts
	chain.Execution.Fills = []execution.FillRecord{
		{Price: price, Quantity: qty, Fee: fee, FeeAsset: "USDT", CostBasis: costBasis, Timestamp: ts},
	}
	chain.Execution.Risk = execution.RiskInput{
		Type:             "rsi_oversold",
		Disposition:      "approved",
		Confidence:       "0.85",
		StrategyType:     "mean_reversion_entry",
		DecisionSeverity: "high",
	}
	return chain
}

func TestGetPairing_Batch_PairedRoundTrip(t *testing.T) {
	now := time.Now()
	// Entry: buy at 50000, exit: sell at 51000 — should produce a paired round-trip with WIN.
	entry := filledChainWithSide("corr-entry-1", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-exit-1", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if result.Summary.PairedCount != 1 {
		t.Errorf("paired_count=%d, want 1", result.Summary.PairedCount)
	}
	if result.Summary.UnmatchedEntries != 0 {
		t.Errorf("unmatched_entries=%d, want 0", result.Summary.UnmatchedEntries)
	}
	if result.Summary.UnmatchedExits != 0 {
		t.Errorf("unmatched_exits=%d, want 0", result.Summary.UnmatchedExits)
	}
	if result.Summary.ResolvedRate != 1.0 {
		t.Errorf("resolved_rate=%f, want 1.0", result.Summary.ResolvedRate)
	}

	// Check attribution.
	if len(result.RoundTrips) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(result.RoundTrips))
	}
	rt := result.RoundTrips[0]
	if rt.Attribution == nil {
		t.Fatal("expected attribution for paired round-trip")
	}
	if rt.Attribution.Outcome != "win" {
		t.Errorf("outcome=%s, want win", rt.Attribution.Outcome)
	}
	// Gross P&L = 5100 - 5000 = 100, fees = 1.0, net = 99.
	if rt.Attribution.GrossPnL != 100.0 {
		t.Errorf("gross_pnl=%f, want 100.0", rt.Attribution.GrossPnL)
	}
}

func TestGetPairing_Batch_UnmatchedEntry(t *testing.T) {
	now := time.Now()
	// Only a buy, no sell — should produce an unmatched entry.
	entry := filledChainWithSide("corr-lone-buy", "buy", "50000.00", "0.1", "0.50", "5000.00", now)

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry},
	}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if result.Summary.PairedCount != 0 {
		t.Errorf("paired_count=%d, want 0", result.Summary.PairedCount)
	}
	if result.Summary.UnmatchedEntries != 1 {
		t.Errorf("unmatched_entries=%d, want 1", result.Summary.UnmatchedEntries)
	}
	if result.Summary.ResolvedRate != 0.0 {
		t.Errorf("resolved_rate=%f, want 0.0", result.Summary.ResolvedRate)
	}

	if len(result.RoundTrips) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(result.RoundTrips))
	}
	if result.RoundTrips[0].Attribution != nil {
		t.Error("expected nil attribution for unmatched entry")
	}
	if string(result.RoundTrips[0].State) != "unmatched_entry" {
		t.Errorf("state=%s, want unmatched_entry", result.RoundTrips[0].State)
	}
}

func TestGetPairing_Batch_StateFilter(t *testing.T) {
	now := time.Now()
	entry := filledChainWithSide("corr-f-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-f-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))
	loner := filledChainWithSide("corr-f-loner", "buy", "49000.00", "0.2", "0.40", "9800.00", now.Add(2*time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit, *loner},
	}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	// Filter to only paired.
	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		State:      "paired",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.RoundTrips) != 1 {
		t.Fatalf("expected 1 paired round-trip, got %d", len(result.RoundTrips))
	}
	if string(result.RoundTrips[0].State) != "paired" {
		t.Errorf("state=%s, want paired", result.RoundTrips[0].State)
	}
}

func TestGetPairing_Batch_RejectedExcluded(t *testing.T) {
	now := time.Now()
	rejected := fullChain("corr-rej")
	rejected.Execution.Status = "rejected"
	rejected.Execution.Timestamp = now

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*rejected},
	}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.RoundTrips) != 0 {
		t.Errorf("expected 0 round-trips for rejected, got %d", len(result.RoundTrips))
	}
	if result.Meta.LegsProduced != 0 {
		t.Errorf("legs_produced=%d, want 0", result.Meta.LegsProduced)
	}
}

func TestGetPairing_Batch_ValidationErrors(t *testing.T) {
	uc := analyticalclient.NewGetPairingUseCase(&stubCompositeReader{}, slog.Default())

	tests := []struct {
		name  string
		query analyticalclient.PairingQuery
	}{
		{"missing source", analyticalclient.PairingQuery{Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"missing symbol", analyticalclient.PairingQuery{Source: "binance", Timeframe: 60}},
		{"invalid timeframe", analyticalclient.PairingQuery{Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), tt.query)
			if prob == nil {
				t.Fatal("expected validation problem")
			}
			if prob.Code != problem.InvalidArgument {
				t.Errorf("expected InvalidArgument, got %s", prob.Code)
			}
		})
	}
}

func TestGetPairing_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetPairingUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		CorrelationID: "x", Instrument: instrumentFromVenue("btcusdt"),
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestGetPairing_Single_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetPairingUseCase(&stubCompositeReader{}, slog.Default())
	_, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		CorrelationID: "corr-001",
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", prob.Code)
	}
}

func TestGetPairing_Single_NoExecution(t *testing.T) {
	chain := fullChain("corr-noexec")
	chain.Execution = nil
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		CorrelationID: "corr-noexec",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.RoundTrips) != 0 {
		t.Errorf("expected 0 round-trips for no execution, got %d", len(result.RoundTrips))
	}
}

func TestGetPairing_Batch_LossRoundTrip(t *testing.T) {
	now := time.Now()
	// Entry: buy at 51000, exit: sell at 50000 — should produce a LOSS.
	entry := filledChainWithSide("corr-loss-entry", "buy", "51000.00", "0.1", "0.50", "5100.00", now)
	exit := filledChainWithSide("corr-loss-exit", "sell", "50000.00", "0.1", "0.50", "5000.00", now.Add(time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetPairingUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.PairingQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.RoundTrips) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(result.RoundTrips))
	}
	if result.RoundTrips[0].Attribution == nil {
		t.Fatal("expected attribution for paired round-trip")
	}
	if result.RoundTrips[0].Attribution.Outcome != "loss" {
		t.Errorf("outcome=%s, want loss", result.RoundTrips[0].Attribution.Outcome)
	}
	if result.Summary.LossCount != 1 {
		t.Errorf("loss_count=%d, want 1", result.Summary.LossCount)
	}
}

// --- Effectiveness with Pairing Integration Tests (S481) ---

func TestGetEffectiveness_Batch_PairedRoundTripProducesWin(t *testing.T) {
	now := time.Now()
	// Buy at 50000, sell at 51000 — ClassifyPair should return win.
	entry := filledChainWithSide("corr-eff-pair-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-eff-pair-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// Should have at least one "win" evaluation from the paired round-trip.
	hasWin := false
	for _, eval := range result.Evaluations {
		if eval.Outcome == "win" {
			hasWin = true
			break
		}
	}
	if !hasWin {
		t.Error("expected at least one win evaluation from paired round-trip")
	}
}

func TestGetEffectiveness_Batch_SingleLegRemainsUnresolved(t *testing.T) {
	now := time.Now()
	// Only a buy, no sell — should remain unresolved.
	entry := filledChainWithSide("corr-eff-single", "buy", "50000.00", "0.1", "0.50", "5000.00", now)

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry},
	}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(result.Evaluations))
	}
	if result.Evaluations[0].Outcome != "unresolved" {
		t.Errorf("outcome=%s, want unresolved for single-leg fill", result.Evaluations[0].Outcome)
	}
}

func TestGetEffectivenessSummary_PairingIntegration_ReducesUnresolved(t *testing.T) {
	now := time.Now()
	entry := filledChainWithSide("corr-summ-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-summ-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))
	loner := filledChainWithSide("corr-summ-loner", "buy", "49000.00", "0.2", "0.40", "9800.00", now.Add(2*time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit, *loner},
	}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Cohorts) != 1 {
		t.Fatalf("expected 1 cohort, got %d", len(result.Cohorts))
	}
	cs := result.Cohorts[0]

	// The paired round-trip should classify as win, the lone buy as unresolved.
	if cs.WinCount != 1 {
		t.Errorf("win_count=%d, want 1 (from paired round-trip)", cs.WinCount)
	}
	if cs.UnresolvedCount != 1 {
		t.Errorf("unresolved_count=%d, want 1 (lone buy)", cs.UnresolvedCount)
	}
	if cs.Evaluated != 2 {
		t.Errorf("evaluated=%d, want 2", cs.Evaluated)
	}
}
