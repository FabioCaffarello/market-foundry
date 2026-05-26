package pairing_test

import (
	"testing"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/execution"
	"internal/domain/instrument"
	"internal/domain/pairing"
)

func btcUSDTPerpExternal(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// ---------------------------------------------------------------------------
// S500: Lifecycle close hardening — pairing edge case tests
// ---------------------------------------------------------------------------

// --- Cancelled-with-partial-fill carry-forward eligibility ---

func TestClassifyCarryForward_CancelledWithFills_IsEligible(t *testing.T) {
	// An order cancelled after partial fill should be eligible (R-CF5).
	intent := execution.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "binance_spot",
		Instrument: btcUSDTPerpExternal(t),
		Timeframe:  60,
		Side:       execution.SideBuy,
		Quantity:   "0.1",
		Status:     execution.StatusCancelled,
		Fills: []execution.FillRecord{
			{
				Price:     "50000",
				Quantity:  "0.05",
				Fee:       "0.025",
				CostBasis: "2500",
				Timestamp: time.Now().UTC(),
			},
		},
		CorrelationID: "corr-cancelled-partial",
		Timestamp:     time.Now().UTC(),
	}

	if got := pairing.ClassifyCarryForward(intent); got != pairing.CarryEligible {
		t.Errorf("cancelled-with-fill: got %s, want %s", got, pairing.CarryEligible)
	}
}

// --- Non-terminal at close: all non-terminal statuses are ineligible ---

func TestClassifyCarryForward_AllNonTerminalStatuses(t *testing.T) {
	nonTerminal := []execution.Status{
		execution.StatusSubmitted,
		execution.StatusSent,
		execution.StatusAccepted,
		execution.StatusPartiallyFilled,
	}

	for _, st := range nonTerminal {
		intent := execution.ExecutionIntent{
			Type:       "venue_market_order",
			Source:     "binance_spot",
			Instrument: btcUSDTPerpExternal(t),
			Timeframe:  60,
			Side:       execution.SideBuy,
			Quantity:   "0.1",
			Status:     st,
			Fills: []execution.FillRecord{
				{Price: "50000", Quantity: "0.05", Fee: "0.025", CostBasis: "2500", Timestamp: time.Now()},
			},
			CorrelationID: "corr-nonterminal",
			Timestamp:     time.Now().UTC(),
		}

		got := pairing.ClassifyCarryForward(intent)
		if got != pairing.CarryIneligibleNonTerminal {
			t.Errorf("status=%s: got %s, want %s", st, got, pairing.CarryIneligibleNonTerminal)
		}
	}
}

// --- Cross-session partial remainder cascade ---

func TestMatchFIFO_CrossSession_PartialRemainderCascade(t *testing.T) {
	// Session 1: entry buy 0.3 at t0
	// Session 2: exit sell 0.1 at t0+1h (partial match)
	// Session 3: exit sell 0.1 at t0+2h (pairs with remainder)
	// Result: 2 paired (0.1 each), 1 unmatched entry (0.1 remainder)
	t0 := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)

	legSet := pairing.CrossSessionLegSet{
		Sessions: []string{"s1", "s2", "s3"},
		Legs: []pairing.SessionLeg{
			{
				Leg: pairing.Leg{
					Direction: pairing.LegEntry, Side: execution.SideBuy,
					Symbol: "BTCUSDT", Source: "binance_spot", Timeframe: 60,
					CorrelationID: "entry-big",
					Price:         "50000", Quantity: "0.3", Fee: "0.15", CostBasis: "15000",
					Timestamp: t0,
				},
				SessionID: "s1",
			},
			{
				Leg: pairing.Leg{
					Direction: pairing.LegExit, Side: execution.SideSell,
					Symbol: "BTCUSDT", Source: "binance_spot", Timeframe: 60,
					CorrelationID: "exit-1",
					Price:         "51000", Quantity: "0.1", Fee: "0.051", CostBasis: "5100",
					Timestamp: t0.Add(time.Hour),
				},
				SessionID: "s2",
			},
			{
				Leg: pairing.Leg{
					Direction: pairing.LegExit, Side: execution.SideSell,
					Symbol: "BTCUSDT", Source: "binance_spot", Timeframe: 60,
					CorrelationID: "exit-2",
					Price:         "52000", Quantity: "0.1", Fee: "0.052", CostBasis: "5200",
					Timestamp: t0.Add(2 * time.Hour),
				},
				SessionID: "s3",
			},
		},
	}

	legs := legSet.ExtractLegs()
	roundTrips := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())
	idx := legSet.SessionLegIndex()
	annotated := pairing.AnnotateRoundTrips(roundTrips, idx)

	var paired, unmatched int
	for _, csrt := range annotated {
		switch csrt.State {
		case pairing.StatePaired:
			paired++
			if !csrt.CrossSession {
				t.Error("expected cross-session for paired round-trips across sessions")
			}
		case pairing.StateUnmatchedEntry:
			unmatched++
		}
	}

	if paired != 2 {
		t.Errorf("paired = %d, want 2", paired)
	}
	if unmatched != 1 {
		t.Errorf("unmatched = %d, want 1 (remainder)", unmatched)
	}

	// Summary should reflect cascade correctly.
	summary := pairing.SummarizeContinuity(annotated)
	if summary.CrossSessionPairedCount != 2 {
		t.Errorf("CrossSessionPairedCount = %d, want 2", summary.CrossSessionPairedCount)
	}
	if summary.OpenCount != 1 {
		t.Errorf("OpenCount = %d, want 1", summary.OpenCount)
	}
}

// --- Boundary timestamp equality ---

func TestMatchFIFO_SameTimestamp_EntryAndExit_Pair(t *testing.T) {
	// When entry and exit have identical timestamps (M4: entry <= exit),
	// they should still pair.
	ts := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)

	legs := []pairing.Leg{
		{
			Direction: pairing.LegEntry, Side: execution.SideBuy,
			Symbol: "BTCUSDT", Source: "binance_spot", Timeframe: 60,
			Price: "50000", Quantity: "0.1", Fee: "0.05", CostBasis: "5000",
			Timestamp: ts,
		},
		{
			Direction: pairing.LegExit, Side: execution.SideSell,
			Symbol: "BTCUSDT", Source: "binance_spot", Timeframe: 60,
			Price: "50100", Quantity: "0.1", Fee: "0.05", CostBasis: "5010",
			Timestamp: ts, // same timestamp
		},
	}

	rts := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())
	if len(rts) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(rts))
	}
	if rts[0].State != pairing.StatePaired {
		t.Errorf("state = %s, want paired (same timestamp should satisfy M4: entry <= exit)", rts[0].State)
	}
}

// --- Halted session origin flag ---

func TestReconcileCrossSession_HaltedSessionOrigin(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_A",
		CrossSession:   false,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:        effectiveness.OutcomeWin,
		GrossPnL:       10,
		NetPnL:         9,
		TotalFees:      1,
		EntryCostBasis: 100,
	}

	lcCtx := &pairing.LifecycleCloseContext{
		EntrySessionHalted: true,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr, lcCtx)

	if !result.HaltedOrigin {
		t.Error("expected HaltedOrigin=true when entry session was halted")
	}

	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}

	if !hasFlag(pairing.FlagHaltedSessionOrigin) {
		t.Error("expected halted_session_origin flag")
	}
	if result.CarryoverReliable {
		t.Error("expected carryover_reliable=false when halted origin")
	}
}

// --- Non-terminal at close flag ---

func TestReconcileCrossSession_NonTerminalAtClose(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_A",
		CrossSession:   false,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:        effectiveness.OutcomeWin,
		GrossPnL:       10,
		NetPnL:         9,
		TotalFees:      1,
		EntryCostBasis: 100,
	}

	lcCtx := &pairing.LifecycleCloseContext{
		ExitNonTerminalAtClose: true,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr, lcCtx)

	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}

	if !hasFlag(pairing.FlagNonTerminalAtClose) {
		t.Error("expected non_terminal_at_close flag")
	}
	if result.CarryoverReliable {
		t.Error("expected carryover_reliable=false when non-terminal at close")
	}
}

// --- No lifecycle close context (nil) — backward compatibility ---

func TestReconcileCrossSession_NilLifecycleContext_BackwardCompatible(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_A",
		CrossSession:   false,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:        effectiveness.OutcomeWin,
		GrossPnL:       10,
		NetPnL:         9,
		TotalFees:      1,
		EntryCostBasis: 100,
	}

	// No lifecycle context — should work exactly as before S500.
	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)

	if !result.Clean {
		t.Errorf("expected clean=true for intra-session with no lifecycle context, got flags=%v", result.Flags)
	}
	if !result.CarryoverReliable {
		t.Error("expected carryover_reliable=true")
	}
	if result.HaltedOrigin {
		t.Error("expected HaltedOrigin=false when no lifecycle context")
	}
}

// --- Combined: halted + non-terminal ---

func TestReconcileCrossSession_HaltedAndNonTerminal(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_B",
		CrossSession:   true,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:        effectiveness.OutcomeWin,
		GrossPnL:       10,
		NetPnL:         9,
		TotalFees:      1,
		EntryCostBasis: 100,
	}

	lcCtx := &pairing.LifecycleCloseContext{
		EntrySessionHalted:     true,
		ExitNonTerminalAtClose: true,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr, lcCtx)

	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}

	if !hasFlag(pairing.FlagHaltedSessionOrigin) {
		t.Error("expected halted_session_origin flag")
	}
	if !hasFlag(pairing.FlagNonTerminalAtClose) {
		t.Error("expected non_terminal_at_close flag")
	}
	if !hasFlag(pairing.FlagCrossSession) {
		t.Error("expected cross_session flag")
	}
	if !hasFlag(pairing.FlagBoundaryCarryover) {
		t.Error("expected boundary_carryover flag")
	}
	if result.CarryoverReliable {
		t.Error("expected carryover_reliable=false for halted + non-terminal")
	}
	if !result.HaltedOrigin {
		t.Error("expected HaltedOrigin=true")
	}
}

// --- Continuity classification: orphan exit from halted session ---

func TestClassifyContinuity_OrphanExitFromHaltedSession_IsGenuineUnresolved(t *testing.T) {
	// An orphan exit is genuine_unresolved regardless of session status.
	// The halted_session_origin flag provides additional context but does
	// not change the continuity classification.
	rt := pairing.RoundTrip{
		Exit:            &pairing.Leg{Side: "sell", Price: "100", Quantity: "1", CostBasis: "100", Timestamp: time.Now()},
		State:           pairing.StateUnmatchedExit,
		UnmatchedReason: pairing.ReasonNoEntryFound,
		Symbol:          "BTCUSDT",
		Source:          "binance_spot",
	}

	continuity := pairing.ClassifyContinuity(rt)
	if continuity != pairing.ContinuityGenuineUnresolved {
		t.Errorf("orphan exit continuity = %s, want genuine_unresolved", continuity)
	}
}

// --- Quantity mismatch remainder continuity ---

func TestClassifyContinuity_QuantityRemainder_IsOpen(t *testing.T) {
	rt := pairing.RoundTrip{
		Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "0.05", CostBasis: "5", Timestamp: time.Now()},
		State:           pairing.StateUnmatchedEntry,
		UnmatchedReason: pairing.ReasonQuantityMismatchResidue,
		Symbol:          "BTCUSDT",
		Source:          "binance_spot",
	}

	continuity := pairing.ClassifyContinuity(rt)
	if continuity != pairing.ContinuityOpen {
		t.Errorf("quantity remainder continuity = %s, want open", continuity)
	}
}

// --- Edge: IntentToLeg for cancelled-with-partial-fill ---

func TestIntentToLeg_CancelledWithPartialFill_ProducesValidLeg(t *testing.T) {
	intent := execution.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "binance_spot",
		Instrument: btcUSDTPerpExternal(t),
		Timeframe:  60,
		Side:       execution.SideBuy,
		Quantity:   "0.1",
		Status:     execution.StatusCancelled,
		Risk:       execution.RiskInput{Type: "ema_crossover", Disposition: "approved"},
		Fills: []execution.FillRecord{
			{Price: "50000", Quantity: "0.03", Fee: "0.015", CostBasis: "1500", Timestamp: time.Now()},
			{Price: "50100", Quantity: "0.02", Fee: "0.010", CostBasis: "1002", Timestamp: time.Now()},
		},
		CorrelationID: "corr-cancel-partial",
		Timestamp:     time.Now().UTC(),
	}

	leg := pairing.IntentToLeg(intent, "long")

	if leg.Direction != pairing.LegEntry {
		t.Errorf("direction = %s, want entry", leg.Direction)
	}
	// Quantity should be aggregated from fills (0.03 + 0.02 = 0.05), not original 0.1.
	if leg.Quantity != "0.05000000" {
		t.Errorf("quantity = %s, want 0.05000000 (aggregated fills)", leg.Quantity)
	}
	if leg.Fee != "0.02500000" {
		t.Errorf("fee = %s, want 0.02500000", leg.Fee)
	}
	if leg.CorrelationID != "corr-cancel-partial" {
		t.Errorf("correlation_id = %s, want corr-cancel-partial", leg.CorrelationID)
	}
}
