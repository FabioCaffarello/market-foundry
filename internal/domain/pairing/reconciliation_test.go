package pairing

import (
	"testing"

	"internal/domain/effectiveness"
	"internal/domain/execution"
)

func TestReconcileRoundTrip_CleanPair(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Price: "50000.00", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5000.00",
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Price: "51000.00", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5100.00",
		},
		State:           StatePaired,
		MatchedQuantity: "0.10000000",
		Instrument:      btcUSDTSpot,
		Source:          "binance_spot",
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeWin,
		GrossPnL: 100.0, NetPnL: 99.0, TotalFees: 1.0,
	}

	result := ReconcileRoundTrip(rt, attr)

	if !result.Clean {
		t.Errorf("expected clean=true, got false; flags=%v", result.Flags)
	}
	if !result.FeeReliable {
		t.Error("expected fee_reliable=true")
	}
	if !result.PnLReliable {
		t.Error("expected pnl_reliable=true")
	}
	if len(result.Flags) != 0 {
		t.Errorf("expected 0 flags, got %d: %v", len(result.Flags), result.Flags)
	}
}

func TestReconcileRoundTrip_FeeGap(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTPerp, Source: "binance_futures",
			Quantity: "0.1", Price: "50000.00", Fee: "0", CostBasis: "5000.00",
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTPerp, Source: "binance_futures",
			Quantity: "0.1", Price: "51000.00", Fee: "0", CostBasis: "5100.00",
		},
		State:      StatePaired,
		Instrument: btcUSDTPerp,
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeWin,
		GrossPnL: 100.0, NetPnL: 100.0, TotalFees: 0,
	}

	result := ReconcileRoundTrip(rt, attr)

	if result.Clean {
		t.Error("expected clean=false for fee gap")
	}
	if result.FeeReliable {
		t.Error("expected fee_reliable=false for zero fees")
	}
	assertHasFlag(t, result, FlagFeeGap)
}

func TestReconcileRoundTrip_CostBasisZero(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Price: "0", Fee: "0", CostBasis: "0",
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Price: "0", Fee: "0", CostBasis: "0",
		},
		State:      StatePaired,
		Instrument: btcUSDTSpot,
	}
	attr := &effectiveness.Attribution{Outcome: effectiveness.OutcomeUnresolved}

	result := ReconcileRoundTrip(rt, attr)

	assertHasFlag(t, result, FlagCostBasisZero)
	assertHasFlag(t, result, FlagFeeGap)
	assertHasFlag(t, result, FlagOutcomeUnresolved)
	if result.PnLReliable {
		t.Error("expected pnl_reliable=false for zero cost basis")
	}
}

func TestReconcileRoundTrip_Simulated(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Simulated: true,
			Quantity: "0.1", Fee: "0", CostBasis: "0",
		},
		State: StateUnmatchedEntry,
	}

	result := ReconcileRoundTrip(rt, nil)

	assertHasFlag(t, result, FlagSimulated)
	assertHasFlag(t, result, FlagUnmatchedOpen)
}

func TestReconcileRoundTrip_UnmatchedEntry(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy,
			Quantity: "0.1", Fee: "0.5", CostBasis: "5000.00",
		},
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonNoExitFound,
	}

	result := ReconcileRoundTrip(rt, nil)

	assertHasFlag(t, result, FlagUnmatchedOpen)
	if result.FeeReliable {
		t.Error("expected fee_reliable=false for unmatched entry")
	}
	if result.PnLReliable {
		t.Error("expected pnl_reliable=false for unmatched entry")
	}
}

func TestReconcileRoundTrip_OrphanExit(t *testing.T) {
	rt := RoundTrip{
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell,
			Quantity: "0.1", Fee: "0.5", CostBasis: "5100.00",
		},
		State:           StateUnmatchedExit,
		UnmatchedReason: ReasonNoEntryFound,
	}

	result := ReconcileRoundTrip(rt, nil)

	assertHasFlag(t, result, FlagOrphanExit)
}

func TestReconcileRoundTrip_FeeAssetMismatch(t *testing.T) {
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy,
			Quantity: "0.1", Fee: "0.50", FeeAsset: "BNB", CostBasis: "5000.00",
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell,
			Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5100.00",
		},
		State:      StatePaired,
		Instrument: btcUSDTSpot,
	}
	attr := &effectiveness.Attribution{Outcome: effectiveness.OutcomeWin, GrossPnL: 100, NetPnL: 99, TotalFees: 1}

	result := ReconcileRoundTrip(rt, attr)

	assertHasFlag(t, result, FlagFeeAssetMismatch)
}

// S499: FeeSource-aware reliability tests.

func TestReconcileRoundTrip_FuturesFeeSourceUnavailableIsReliable(t *testing.T) {
	// Futures round-trip with FeeSource=unavailable should be fee_reliable=true
	// because the system knows why fee=0 — it's an expected API limitation.
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTPerp, Source: "binance_futures",
			Quantity: "0.1", Price: "50000.00", Fee: "0", CostBasis: "5000.00",
			FeeSource: execution.FeeSourceUnavailable,
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTPerp, Source: "binance_futures",
			Quantity: "0.1", Price: "51000.00", Fee: "0", CostBasis: "5100.00",
			FeeSource: execution.FeeSourceUnavailable,
		},
		State:      StatePaired,
		Instrument: btcUSDTPerp,
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeWin,
		GrossPnL: 100.0, NetPnL: 100.0, TotalFees: 0,
	}

	result := ReconcileRoundTrip(rt, attr)

	if !result.FeeReliable {
		t.Error("expected fee_reliable=true for FeeSourceUnavailable (Futures)")
	}
	// Fee gap flag should still be present — the fee IS zero, just acknowledged.
	assertHasFlag(t, result, FlagFeeGap)
}

func TestReconcileRoundTrip_FeeRatioAnomaly(t *testing.T) {
	// Fee = 600 on cost_basis = 5000 → 12% ratio → anomaly.
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "600.00", FeeAsset: "USDT", CostBasis: "5000.00",
			FeeSource: execution.FeeSourceVenue,
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5100.00",
			FeeSource: execution.FeeSourceVenue,
		},
		State:      StatePaired,
		Instrument: btcUSDTSpot,
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeLoss,
		GrossPnL: 100.0, NetPnL: -500.5, TotalFees: 600.5,
	}

	result := ReconcileRoundTrip(rt, attr)

	assertHasFlag(t, result, FlagFeeRatioAnomaly)
}

func TestReconcileRoundTrip_FeeRatioNormal(t *testing.T) {
	// Fee = 0.50 on cost_basis = 5000 → 0.01% → normal.
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5000.00",
			FeeSource: execution.FeeSourceVenue,
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5100.00",
			FeeSource: execution.FeeSourceVenue,
		},
		State:      StatePaired,
		Instrument: btcUSDTSpot,
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeWin,
		GrossPnL: 100.0, NetPnL: 99.0, TotalFees: 1.0,
	}

	result := ReconcileRoundTrip(rt, attr)

	assertNoFlag(t, result, FlagFeeRatioAnomaly)
}

func TestReconcileRoundTrip_FeeSourceFallback(t *testing.T) {
	// A leg with FeeSource=fallback should trigger the flag.
	rt := RoundTrip{
		Entry: &Leg{
			Direction: LegEntry, Side: execution.SideBuy, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "0", CostBasis: "5000.00",
			FeeSource: execution.FeeSourceFallback,
		},
		Exit: &Leg{
			Direction: LegExit, Side: execution.SideSell, Instrument: btcUSDTSpot, Source: "binance_spot",
			Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5100.00",
			FeeSource: execution.FeeSourceVenue,
		},
		State:      StatePaired,
		Instrument: btcUSDTSpot,
	}
	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeWin,
		GrossPnL: 100.0, NetPnL: 99.5, TotalFees: 0.5,
	}

	result := ReconcileRoundTrip(rt, attr)

	assertHasFlag(t, result, FlagFeeSourceFallback)
	assertHasFlag(t, result, FlagFeeGap) // entry has fee=0
}

// assertNoFlag checks that a specific flag is NOT present in the result.
func assertNoFlag(t *testing.T, result ReconciliationResult, unexpected ReconciliationFlag) {
	t.Helper()
	for _, f := range result.Flags {
		if f == unexpected {
			t.Errorf("unexpected flag %q found in %v", unexpected, result.Flags)
			return
		}
	}
}

// assertHasFlag checks that a specific flag is present in the result.
func assertHasFlag(t *testing.T, result ReconciliationResult, expected ReconciliationFlag) {
	t.Helper()
	for _, f := range result.Flags {
		if f == expected {
			return
		}
	}
	t.Errorf("expected flag %q not found in %v", expected, result.Flags)
}
