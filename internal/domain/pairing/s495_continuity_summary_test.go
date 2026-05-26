package pairing

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// S495: ContinuitySummary tests
// ---------------------------------------------------------------------------

func TestSummarizeContinuity_Empty(t *testing.T) {
	s := SummarizeContinuity(nil)
	if s.Total != 0 {
		t.Errorf("Total = %d, want 0", s.Total)
	}
	if s.ResolutionRate != 0 {
		t.Errorf("ResolutionRate = %f, want 0", s.ResolutionRate)
	}
}

func TestSummarizeContinuity_AllResolved_IntraSession(t *testing.T) {
	rts := []CrossSessionRoundTrip{
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s1",
			CrossSession:   false,
			Continuity:     ContinuityResolved,
		},
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s1",
			CrossSession:   false,
			Continuity:     ContinuityResolved,
		},
	}

	s := SummarizeContinuity(rts)
	if s.Total != 2 {
		t.Errorf("Total = %d, want 2", s.Total)
	}
	if s.ResolvedCount != 2 {
		t.Errorf("ResolvedCount = %d, want 2", s.ResolvedCount)
	}
	if s.IntraSessionPairedCount != 2 {
		t.Errorf("IntraSessionPairedCount = %d, want 2", s.IntraSessionPairedCount)
	}
	if s.CrossSessionPairedCount != 0 {
		t.Errorf("CrossSessionPairedCount = %d, want 0", s.CrossSessionPairedCount)
	}
	if s.ResolutionRate != 1.0 {
		t.Errorf("ResolutionRate = %f, want 1.0", s.ResolutionRate)
	}
}

func TestSummarizeContinuity_CrossSessionPairs(t *testing.T) {
	rts := []CrossSessionRoundTrip{
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s2",
			CrossSession:   true,
			Continuity:     ContinuityResolved,
		},
		{
			RoundTrip:      RoundTrip{State: StateUnmatchedEntry, UnmatchedReason: ReasonSessionBoundary},
			EntrySessionID: "s2",
			CrossSession:   false,
			Continuity:     ContinuityArtificialUnresolved,
		},
	}

	s := SummarizeContinuity(rts)
	if s.Total != 2 {
		t.Errorf("Total = %d, want 2", s.Total)
	}
	if s.ResolvedCount != 1 {
		t.Errorf("ResolvedCount = %d, want 1", s.ResolvedCount)
	}
	if s.CrossSessionPairedCount != 1 {
		t.Errorf("CrossSessionPairedCount = %d, want 1", s.CrossSessionPairedCount)
	}
	if s.ArtificialUnresolvedCount != 1 {
		t.Errorf("ArtificialUnresolvedCount = %d, want 1", s.ArtificialUnresolvedCount)
	}
	// CrossSessionResolutionRate = 1 / (1 + 1) = 0.5
	if s.CrossSessionResolutionRate != 0.5 {
		t.Errorf("CrossSessionResolutionRate = %f, want 0.5", s.CrossSessionResolutionRate)
	}
}

func TestSummarizeContinuity_MixedStates(t *testing.T) {
	rts := []CrossSessionRoundTrip{
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s1",
			CrossSession:   false,
			Continuity:     ContinuityResolved,
		},
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s2",
			CrossSession:   true,
			Continuity:     ContinuityResolved,
		},
		{
			RoundTrip:      RoundTrip{State: StateUnmatchedEntry, UnmatchedReason: ReasonNoExitFound},
			EntrySessionID: "s2",
			Continuity:     ContinuityOpen,
		},
		{
			RoundTrip:      RoundTrip{State: StateUnmatchedEntry, UnmatchedReason: ReasonRejectedLeg},
			EntrySessionID: "s1",
			Continuity:     ContinuityGenuineUnresolved,
		},
		{
			RoundTrip:     RoundTrip{State: StateUnmatchedExit, UnmatchedReason: ReasonNoEntryFound},
			ExitSessionID: "s2",
			Continuity:    ContinuityGenuineUnresolved,
		},
	}

	s := SummarizeContinuity(rts)
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.ResolvedCount != 2 {
		t.Errorf("ResolvedCount = %d, want 2", s.ResolvedCount)
	}
	if s.OpenCount != 1 {
		t.Errorf("OpenCount = %d, want 1", s.OpenCount)
	}
	if s.GenuineUnresolvedCount != 2 {
		t.Errorf("GenuineUnresolvedCount = %d, want 2", s.GenuineUnresolvedCount)
	}
	if s.IntraSessionPairedCount != 1 {
		t.Errorf("IntraSessionPairedCount = %d, want 1", s.IntraSessionPairedCount)
	}
	if s.CrossSessionPairedCount != 1 {
		t.Errorf("CrossSessionPairedCount = %d, want 1", s.CrossSessionPairedCount)
	}
	// ResolutionRate = 2/5 = 0.4
	wantRate := 2.0 / 5.0
	if s.ResolutionRate != wantRate {
		t.Errorf("ResolutionRate = %f, want %f", s.ResolutionRate, wantRate)
	}
}

func TestSummarizeContinuity_AllOpen(t *testing.T) {
	rts := []CrossSessionRoundTrip{
		{
			RoundTrip:  RoundTrip{State: StateUnmatchedEntry, UnmatchedReason: ReasonNoExitFound},
			Continuity: ContinuityOpen,
		},
		{
			RoundTrip:  RoundTrip{State: StateUnmatchedEntry, UnmatchedReason: ReasonQuantityMismatchResidue},
			Continuity: ContinuityOpen,
		},
	}

	s := SummarizeContinuity(rts)
	if s.ResolvedCount != 0 {
		t.Errorf("ResolvedCount = %d, want 0", s.ResolvedCount)
	}
	if s.OpenCount != 2 {
		t.Errorf("OpenCount = %d, want 2", s.OpenCount)
	}
	if s.ResolutionRate != 0 {
		t.Errorf("ResolutionRate = %f, want 0", s.ResolutionRate)
	}
	if s.CrossSessionResolutionRate != 0 {
		t.Errorf("CrossSessionResolutionRate = %f, want 0", s.CrossSessionResolutionRate)
	}
}

func TestSummarizeContinuity_FullCrossSessionResolution(t *testing.T) {
	// All artificial_unresolved were resolved by cross-session matching.
	rts := []CrossSessionRoundTrip{
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s1",
			ExitSessionID:  "s2",
			CrossSession:   true,
			Continuity:     ContinuityResolved,
		},
		{
			RoundTrip:      RoundTrip{State: StatePaired},
			EntrySessionID: "s2",
			ExitSessionID:  "s3",
			CrossSession:   true,
			Continuity:     ContinuityResolved,
		},
	}

	s := SummarizeContinuity(rts)
	if s.CrossSessionPairedCount != 2 {
		t.Errorf("CrossSessionPairedCount = %d, want 2", s.CrossSessionPairedCount)
	}
	if s.ArtificialUnresolvedCount != 0 {
		t.Errorf("ArtificialUnresolvedCount = %d, want 0", s.ArtificialUnresolvedCount)
	}
	// CrossSessionResolutionRate = 2 / (2 + 0) = 1.0
	if s.CrossSessionResolutionRate != 1.0 {
		t.Errorf("CrossSessionResolutionRate = %f, want 1.0", s.CrossSessionResolutionRate)
	}
}

// ---------------------------------------------------------------------------
// S495: CrossSessionLegSet additional tests
// ---------------------------------------------------------------------------

func TestS495_CrossSessionLegSet_ExtractLegs_PreservesOrder(t *testing.T) {
	t1 := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	ls := CrossSessionLegSet{
		Legs: []SessionLeg{
			{Leg: Leg{Timestamp: t1, CorrelationID: "c1"}, SessionID: "s1"},
			{Leg: Leg{Timestamp: t2, CorrelationID: "c2"}, SessionID: "s1"},
			{Leg: Leg{Timestamp: t3, CorrelationID: "c3"}, SessionID: "s2"},
		},
	}

	legs := ls.ExtractLegs()
	if len(legs) != 3 {
		t.Fatalf("ExtractLegs: got %d legs, want 3", len(legs))
	}
	if legs[0].CorrelationID != "c1" || legs[1].CorrelationID != "c2" || legs[2].CorrelationID != "c3" {
		t.Error("ExtractLegs did not preserve order")
	}
}

func TestS495_CrossSessionLegSet_SessionLegIndex(t *testing.T) {
	ls := CrossSessionLegSet{
		Legs: []SessionLeg{
			{Leg: Leg{CorrelationID: "c1"}, SessionID: "s1"},
			{Leg: Leg{CorrelationID: "c2"}, SessionID: "s2"},
		},
	}

	idx := ls.SessionLegIndex()
	if len(idx) != 2 {
		t.Fatalf("SessionLegIndex: got %d entries, want 2", len(idx))
	}
	if idx["c1"].SessionID != "s1" {
		t.Errorf("c1 session = %s, want s1", idx["c1"].SessionID)
	}
	if idx["c2"].SessionID != "s2" {
		t.Errorf("c2 session = %s, want s2", idx["c2"].SessionID)
	}
}

// ---------------------------------------------------------------------------
// S495: AnnotateRoundTrips cross-session provenance tests
// ---------------------------------------------------------------------------

func TestAnnotateRoundTrips_CrossSessionProvenance(t *testing.T) {
	t1 := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	rts := []RoundTrip{
		{
			Entry:           &Leg{CorrelationID: "c1", Timestamp: t1},
			Exit:            &Leg{CorrelationID: "c2", Timestamp: t2},
			State:           StatePaired,
			MatchedQuantity: "0.1",
			Instrument:      btcUSDTSpot,
			Source:          "binance_spot",
		},
	}

	idx := map[string]SessionLeg{
		"c1": {Leg: Leg{CorrelationID: "c1"}, SessionID: "session_1"},
		"c2": {Leg: Leg{CorrelationID: "c2"}, SessionID: "session_2"},
	}

	annotated := AnnotateRoundTrips(rts, idx)
	if len(annotated) != 1 {
		t.Fatalf("got %d annotated, want 1", len(annotated))
	}
	if annotated[0].EntrySessionID != "session_1" {
		t.Errorf("EntrySessionID = %s, want session_1", annotated[0].EntrySessionID)
	}
	if annotated[0].ExitSessionID != "session_2" {
		t.Errorf("ExitSessionID = %s, want session_2", annotated[0].ExitSessionID)
	}
	if !annotated[0].CrossSession {
		t.Error("CrossSession should be true for different sessions")
	}
	if annotated[0].Continuity != ContinuityResolved {
		t.Errorf("Continuity = %s, want resolved", annotated[0].Continuity)
	}
}

func TestAnnotateRoundTrips_ArtificialUnresolved(t *testing.T) {
	rts := []RoundTrip{
		{
			Entry:           &Leg{CorrelationID: "c1"},
			State:           StateUnmatchedEntry,
			UnmatchedReason: ReasonSessionBoundary,
			Instrument:      btcUSDTSpot,
			Source:          "binance_spot",
		},
	}

	idx := map[string]SessionLeg{
		"c1": {Leg: Leg{CorrelationID: "c1"}, SessionID: "session_1"},
	}

	annotated := AnnotateRoundTrips(rts, idx)
	if annotated[0].Continuity != ContinuityArtificialUnresolved {
		t.Errorf("Continuity = %s, want artificial_unresolved", annotated[0].Continuity)
	}
	if annotated[0].CrossSession {
		t.Error("CrossSession should be false for unmatched entry")
	}
}

func TestAnnotateRoundTrips_GenuineUnresolved_Rejected(t *testing.T) {
	rts := []RoundTrip{
		{
			Entry:           &Leg{CorrelationID: "c1"},
			State:           StateUnmatchedEntry,
			UnmatchedReason: ReasonRejectedLeg,
			Instrument:      btcUSDTSpot,
			Source:          "binance_spot",
		},
	}

	idx := map[string]SessionLeg{
		"c1": {Leg: Leg{CorrelationID: "c1"}, SessionID: "session_1"},
	}

	annotated := AnnotateRoundTrips(rts, idx)
	if annotated[0].Continuity != ContinuityGenuineUnresolved {
		t.Errorf("Continuity = %s, want genuine_unresolved", annotated[0].Continuity)
	}
}

func TestAnnotateRoundTrips_Open_NoExitFound(t *testing.T) {
	rts := []RoundTrip{
		{
			Entry:           &Leg{CorrelationID: "c1"},
			State:           StateUnmatchedEntry,
			UnmatchedReason: ReasonNoExitFound,
			Instrument:      btcUSDTSpot,
			Source:          "binance_spot",
		},
	}

	idx := map[string]SessionLeg{
		"c1": {Leg: Leg{CorrelationID: "c1"}, SessionID: "session_1"},
	}

	annotated := AnnotateRoundTrips(rts, idx)
	if annotated[0].Continuity != ContinuityOpen {
		t.Errorf("Continuity = %s, want open", annotated[0].Continuity)
	}
}

// ---------------------------------------------------------------------------
// S495: Integration test — cross-session FIFO matching produces correct continuity
// ---------------------------------------------------------------------------

func TestCrossSession_EndToEnd_TwoSessionsFIFO(t *testing.T) {
	// Session 1: entry buy at t1, no exit
	// Session 2: exit sell at t2
	// Cross-session matching should pair them.
	t1 := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 21, 14, 0, 0, 0, time.UTC)
	s1Close := time.Date(2026, 3, 20, 18, 0, 0, 0, time.UTC)

	legSet := CrossSessionLegSet{
		Window: CrossSessionWindow{
			Symbol:    "BTCUSDT",
			Source:    "binance_spot",
			Timeframe: 60,
			Since:     time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		},
		Sessions: []string{"session_1", "session_2"},
		Legs: []SessionLeg{
			{
				Leg: Leg{
					Direction:     LegEntry,
					Side:          "buy",
					Instrument:    btcUSDTSpot,
					Source:        "binance_spot",
					CorrelationID: "c1",
					Price:         "50000.00000000",
					Quantity:      "0.10000000",
					Fee:           "0.05000000",
					CostBasis:     "5000.00000000",
					Timestamp:     t1,
				},
				SessionID:        "session_1",
				SessionStartedAt: t1.Add(-time.Hour),
				SessionClosedAt:  &s1Close,
			},
			{
				Leg: Leg{
					Direction:     LegExit,
					Side:          "sell",
					Instrument:    btcUSDTSpot,
					Source:        "binance_spot",
					CorrelationID: "c2",
					Price:         "51000.00000000",
					Quantity:      "0.10000000",
					Fee:           "0.05100000",
					CostBasis:     "5100.00000000",
					Timestamp:     t2,
				},
				SessionID:        "session_2",
				SessionStartedAt: t2.Add(-time.Hour),
			},
		},
	}

	// Extract and match.
	plainLegs := legSet.ExtractLegs()
	roundTrips := MatchFIFO(plainLegs, DefaultMatchingConfig())

	if len(roundTrips) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(roundTrips))
	}
	if roundTrips[0].State != StatePaired {
		t.Errorf("State = %s, want paired", roundTrips[0].State)
	}

	// Annotate.
	idx := legSet.SessionLegIndex()
	annotated := AnnotateRoundTrips(roundTrips, idx)

	if len(annotated) != 1 {
		t.Fatalf("expected 1 annotated, got %d", len(annotated))
	}
	if !annotated[0].CrossSession {
		t.Error("expected CrossSession=true")
	}
	if annotated[0].EntrySessionID != "session_1" {
		t.Errorf("EntrySessionID = %s, want session_1", annotated[0].EntrySessionID)
	}
	if annotated[0].ExitSessionID != "session_2" {
		t.Errorf("ExitSessionID = %s, want session_2", annotated[0].ExitSessionID)
	}
	if annotated[0].Continuity != ContinuityResolved {
		t.Errorf("Continuity = %s, want resolved", annotated[0].Continuity)
	}

	// Summarize.
	summary := SummarizeContinuity(annotated)
	if summary.CrossSessionPairedCount != 1 {
		t.Errorf("CrossSessionPairedCount = %d, want 1", summary.CrossSessionPairedCount)
	}
	if summary.ResolutionRate != 1.0 {
		t.Errorf("ResolutionRate = %f, want 1.0", summary.ResolutionRate)
	}
	if summary.CrossSessionResolutionRate != 1.0 {
		t.Errorf("CrossSessionResolutionRate = %f, want 1.0", summary.CrossSessionResolutionRate)
	}
}

func TestCrossSession_EndToEnd_ThreeSessions_MixedOutcomes(t *testing.T) {
	// Session 1: entry (c1), entry (c2)
	// Session 2: exit (c3) pairs with c1, entry (c4) no exit
	// Session 3: exit (c5) pairs with c2
	// Remaining: c4 is open
	t1 := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	t4 := time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC)
	t5 := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)

	legs := []SessionLeg{
		{Leg: Leg{Direction: LegEntry, Side: "buy", Instrument: btcUSDTSpot, Source: "binance_spot", CorrelationID: "c1", Price: "50000", Quantity: "0.1", Fee: "0.05", CostBasis: "5000", Timestamp: t1}, SessionID: "s1"},
		{Leg: Leg{Direction: LegEntry, Side: "buy", Instrument: btcUSDTSpot, Source: "binance_spot", CorrelationID: "c2", Price: "49000", Quantity: "0.1", Fee: "0.049", CostBasis: "4900", Timestamp: t2}, SessionID: "s1"},
		{Leg: Leg{Direction: LegExit, Side: "sell", Instrument: btcUSDTSpot, Source: "binance_spot", CorrelationID: "c3", Price: "51000", Quantity: "0.1", Fee: "0.051", CostBasis: "5100", Timestamp: t3}, SessionID: "s2"},
		{Leg: Leg{Direction: LegEntry, Side: "buy", Instrument: btcUSDTSpot, Source: "binance_spot", CorrelationID: "c4", Price: "52000", Quantity: "0.1", Fee: "0.052", CostBasis: "5200", Timestamp: t4}, SessionID: "s2"},
		{Leg: Leg{Direction: LegExit, Side: "sell", Instrument: btcUSDTSpot, Source: "binance_spot", CorrelationID: "c5", Price: "50000", Quantity: "0.1", Fee: "0.05", CostBasis: "5000", Timestamp: t5}, SessionID: "s3"},
	}

	legSet := CrossSessionLegSet{
		Sessions: []string{"s1", "s2", "s3"},
		Legs:     legs,
	}

	plainLegs := legSet.ExtractLegs()
	roundTrips := MatchFIFO(plainLegs, DefaultMatchingConfig())
	idx := legSet.SessionLegIndex()
	annotated := AnnotateRoundTrips(roundTrips, idx)
	summary := SummarizeContinuity(annotated)

	// c1+c3 paired cross-session (s1→s2), c2+c5 paired cross-session (s1→s3), c4 unmatched open
	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3", summary.Total)
	}
	if summary.ResolvedCount != 2 {
		t.Errorf("ResolvedCount = %d, want 2", summary.ResolvedCount)
	}
	if summary.CrossSessionPairedCount != 2 {
		t.Errorf("CrossSessionPairedCount = %d, want 2", summary.CrossSessionPairedCount)
	}
	if summary.OpenCount != 1 {
		t.Errorf("OpenCount = %d, want 1", summary.OpenCount)
	}
}
