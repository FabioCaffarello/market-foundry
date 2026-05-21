package pairing_test

import (
	"testing"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
)

// ---------------------------------------------------------------------------
// ReconcileCrossSessionRoundTrip — flag coverage
// ---------------------------------------------------------------------------

func TestReconcileCrossSession_IntraSessionPaired_Clean(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "100", Simulated: false, Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Simulated: false, Timestamp: time.Now()},
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
		Outcome:       effectiveness.OutcomeWin,
		GrossPnL:      10,
		NetPnL:        9,
		TotalFees:     1,
		EntryCostBasis: 100,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)

	if !result.Clean {
		t.Errorf("expected clean=true for intra-session paired, got flags=%v", result.Flags)
	}
	if result.CrossSession {
		t.Error("expected cross_session=false")
	}
	if !result.CarryoverReliable {
		t.Error("expected carryover_reliable=true for intra-session with reliable data")
	}
	if result.Continuity != pairing.ContinuityResolved {
		t.Errorf("expected continuity=resolved, got %s", result.Continuity)
	}
}

func TestReconcileCrossSession_CrossSessionPaired_FlagsPresent(t *testing.T) {
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
		Outcome:       effectiveness.OutcomeWin,
		GrossPnL:      10,
		NetPnL:        9,
		TotalFees:     1,
		EntryCostBasis: 100,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)

	if result.Clean {
		t.Error("expected clean=false for cross-session paired (has cross_session + boundary_carryover flags)")
	}
	if !result.CrossSession {
		t.Error("expected cross_session=true")
	}

	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}

	if !hasFlag(pairing.FlagCrossSession) {
		t.Error("expected cross_session flag")
	}
	if !hasFlag(pairing.FlagBoundaryCarryover) {
		t.Error("expected boundary_carryover flag for resolved cross-session pair")
	}
	if result.CarryoverReliable != true {
		t.Error("expected carryover_reliable=true for cross-session pair with full fee/P&L data")
	}
}

func TestReconcileCrossSession_CrossSessionFeeGap(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0", FeeAsset: "USDT", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0.5", FeeAsset: "USDT", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_futures",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_B",
		CrossSession:   true,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:       effectiveness.OutcomeWin,
		GrossPnL:      10,
		NetPnL:        9.5,
		TotalFees:     0.5,
		EntryCostBasis: 100,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)

	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}

	if !hasFlag(pairing.FlagCrossSessionFeeGap) {
		t.Error("expected cross_session_fee_gap flag when entry fee is zero")
	}
	if !hasFlag(pairing.FlagFeeGap) {
		t.Error("expected fee_gap base flag when entry fee is zero")
	}
	if result.CarryoverReliable {
		t.Error("expected carryover_reliable=false when cross-session fee gap exists")
	}
}

func TestReconcileCrossSession_UnmatchedEntry_Open(t *testing.T) {
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry: &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0.5", CostBasis: "100", Timestamp: time.Now()},
			State: pairing.StateUnmatchedEntry,
		},
		EntrySessionID: "session_A",
		CrossSession:   false,
		Continuity:     pairing.ContinuityOpen,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, nil)

	if result.Continuity != pairing.ContinuityOpen {
		t.Errorf("expected continuity=open, got %s", result.Continuity)
	}
	hasFlag := func(f pairing.ReconciliationFlag) bool {
		for _, fl := range result.Flags {
			if fl == f {
				return true
			}
		}
		return false
	}
	if !hasFlag(pairing.FlagUnmatchedOpen) {
		t.Error("expected unmatched_open flag")
	}
	if result.CarryoverReliable {
		t.Error("expected carryover_reliable=false for unmatched entry")
	}
}

// ---------------------------------------------------------------------------
// SummarizeContinuityReconciliation
// ---------------------------------------------------------------------------

func TestSummarizeContinuityReconciliation_Empty(t *testing.T) {
	s := pairing.SummarizeContinuityReconciliation(nil)
	if s.Total != 0 {
		t.Errorf("expected total=0, got %d", s.Total)
	}
	if len(s.FlagCounts) != 0 {
		t.Errorf("expected empty flag counts, got %v", s.FlagCounts)
	}
}

func TestSummarizeContinuityReconciliation_MixedResults(t *testing.T) {
	results := []pairing.ContinuityReconciliationResult{
		{
			ReconciliationResult: pairing.ReconciliationResult{
				Flags:       nil,
				Clean:       true,
				FeeReliable: true,
				PnLReliable: true,
			},
			Continuity:        pairing.ContinuityResolved,
			CrossSession:      false,
			CarryoverReliable: true,
		},
		{
			ReconciliationResult: pairing.ReconciliationResult{
				Flags:       []pairing.ReconciliationFlag{pairing.FlagCrossSession, pairing.FlagBoundaryCarryover},
				Clean:       false,
				FeeReliable: true,
				PnLReliable: true,
			},
			Continuity:        pairing.ContinuityResolved,
			CrossSession:      true,
			CarryoverReliable: true,
		},
		{
			ReconciliationResult: pairing.ReconciliationResult{
				Flags:       []pairing.ReconciliationFlag{pairing.FlagCrossSession, pairing.FlagCrossSessionFeeGap, pairing.FlagFeeGap},
				Clean:       false,
				FeeReliable: false,
				PnLReliable: false,
			},
			Continuity:        pairing.ContinuityResolved,
			CrossSession:      true,
			CarryoverReliable: false,
		},
	}

	s := pairing.SummarizeContinuityReconciliation(results)

	if s.Total != 3 {
		t.Errorf("expected total=3, got %d", s.Total)
	}
	if s.CleanCount != 1 {
		t.Errorf("expected clean_count=1, got %d", s.CleanCount)
	}
	if s.FlaggedCount != 2 {
		t.Errorf("expected flagged_count=2, got %d", s.FlaggedCount)
	}
	if s.CrossSessionCount != 2 {
		t.Errorf("expected cross_session_count=2, got %d", s.CrossSessionCount)
	}
	if s.BoundaryCarryoverCount != 1 {
		t.Errorf("expected boundary_carryover_count=1, got %d", s.BoundaryCarryoverCount)
	}
	if s.CarryoverReliableCount != 2 {
		t.Errorf("expected carryover_reliable_count=2, got %d", s.CarryoverReliableCount)
	}
	if s.FeeReliableCount != 2 {
		t.Errorf("expected fee_reliable_count=2, got %d", s.FeeReliableCount)
	}
	if s.PnLReliableCount != 2 {
		t.Errorf("expected pnl_reliable_count=2, got %d", s.PnLReliableCount)
	}
	if s.FlagCounts["cross_session"] != 2 {
		t.Errorf("expected cross_session flag count=2, got %d", s.FlagCounts["cross_session"])
	}
	if s.FlagCounts["boundary_carryover"] != 1 {
		t.Errorf("expected boundary_carryover flag count=1, got %d", s.FlagCounts["boundary_carryover"])
	}
}

// ---------------------------------------------------------------------------
// appendIfAbsent (via ReconcileCrossSessionRoundTrip behavior)
// ---------------------------------------------------------------------------

func TestReconcileCrossSession_NoDuplicateFlags(t *testing.T) {
	// A cross-session pair with fee gap already gets FlagFeeGap from base reconciliation.
	// Cross-session reconciliation should add FlagCrossSessionFeeGap but not duplicate FlagFeeGap.
	csrt := pairing.CrossSessionRoundTrip{
		RoundTrip: pairing.RoundTrip{
			Entry:           &pairing.Leg{Side: "buy", Price: "100", Quantity: "1", Fee: "0", CostBasis: "100", Timestamp: time.Now()},
			Exit:            &pairing.Leg{Side: "sell", Price: "110", Quantity: "1", Fee: "0", CostBasis: "110", Timestamp: time.Now()},
			State:           pairing.StatePaired,
			MatchedQuantity: "1",
			Symbol:          "BTCUSDT",
			Source:          "binance_futures",
		},
		EntrySessionID: "session_A",
		ExitSessionID:  "session_B",
		CrossSession:   true,
		Continuity:     pairing.ContinuityResolved,
	}

	attr := &effectiveness.Attribution{
		Outcome:  effectiveness.OutcomeUnresolved,
		GrossPnL: 10,
	}

	result := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)

	// Count occurrences of each flag.
	counts := make(map[pairing.ReconciliationFlag]int)
	for _, f := range result.Flags {
		counts[f]++
	}

	for flag, count := range counts {
		if count > 1 {
			t.Errorf("flag %q appears %d times, expected at most 1", flag, count)
		}
	}
}
