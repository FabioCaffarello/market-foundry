package pairing

import (
	"testing"
	"time"

	"internal/domain/execution"
)

// ---------------------------------------------------------------------------
// S494: Cross-session continuity model tests
// ---------------------------------------------------------------------------

// --- ContinuityState validation ---

func TestValidContinuityState(t *testing.T) {
	tests := []struct {
		cs    ContinuityState
		valid bool
	}{
		{ContinuityResolved, true},
		{ContinuityOpen, true},
		{ContinuityGenuineUnresolved, true},
		{ContinuityArtificialUnresolved, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidContinuityState(tt.cs); got != tt.valid {
			t.Errorf("ValidContinuityState(%q) = %v, want %v", tt.cs, got, tt.valid)
		}
	}
}

// --- CarryForwardEligibility validation ---

func TestValidCarryForwardEligibility(t *testing.T) {
	tests := []struct {
		e     CarryForwardEligibility
		valid bool
	}{
		{CarryEligible, true},
		{CarryIneligibleRejected, true},
		{CarryIneligibleCancelled, true},
		{CarryIneligibleNonTerminal, true},
		{CarryIneligibleNoFills, true},
		{CarryIneligibleAlreadyPaired, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidCarryForwardEligibility(tt.e); got != tt.valid {
			t.Errorf("ValidCarryForwardEligibility(%q) = %v, want %v", tt.e, got, tt.valid)
		}
	}
}

// --- ClassifyCarryForward tests (R-CF1 through R-CF5) ---

func TestClassifyCarryForward_RejectedIsIneligible(t *testing.T) {
	intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
	intent.Status = execution.StatusRejected
	intent.Fills = nil
	if got := ClassifyCarryForward(intent); got != CarryIneligibleRejected {
		t.Errorf("rejected: got %s, want %s", got, CarryIneligibleRejected)
	}
}

func TestClassifyCarryForward_NonTerminalIsIneligible(t *testing.T) {
	for _, status := range []execution.Status{
		execution.StatusSubmitted,
		execution.StatusSent,
		execution.StatusAccepted,
	} {
		intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
		intent.Status = status
		if got := ClassifyCarryForward(intent); got != CarryIneligibleNonTerminal {
			t.Errorf("status=%s: got %s, want %s", status, got, CarryIneligibleNonTerminal)
		}
	}
}

func TestClassifyCarryForward_CancelledNoFillsIsIneligible(t *testing.T) {
	intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
	intent.Status = execution.StatusCancelled
	intent.Fills = nil
	if got := ClassifyCarryForward(intent); got != CarryIneligibleCancelled {
		t.Errorf("cancelled-no-fill: got %s, want %s", got, CarryIneligibleCancelled)
	}
}

func TestClassifyCarryForward_FilledNoFillsIsIneligible(t *testing.T) {
	intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
	intent.Fills = nil
	if got := ClassifyCarryForward(intent); got != CarryIneligibleNoFills {
		t.Errorf("filled-no-fills: got %s, want %s", got, CarryIneligibleNoFills)
	}
}

func TestClassifyCarryForward_FilledWithFillsIsEligible(t *testing.T) {
	intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
	if got := ClassifyCarryForward(intent); got != CarryEligible {
		t.Errorf("filled-with-fills: got %s, want %s", got, CarryEligible)
	}
}

func TestClassifyCarryForward_PartiallyFilledIsIneligible(t *testing.T) {
	intent := makeFilledIntent(t, execution.SideBuy, "50000", "0.1")
	intent.Status = execution.StatusPartiallyFilled
	// partially_filled is non-terminal
	if got := ClassifyCarryForward(intent); got != CarryIneligibleNonTerminal {
		t.Errorf("partially_filled: got %s, want %s", got, CarryIneligibleNonTerminal)
	}
}

// --- ClassifyContinuity tests (C-1 through C-6) ---

func TestClassifyContinuity_PairedIsResolved(t *testing.T) {
	rt := RoundTrip{State: StatePaired}
	if got := ClassifyContinuity(rt); got != ContinuityResolved {
		t.Errorf("paired: got %s, want %s", got, ContinuityResolved)
	}
}

func TestClassifyContinuity_SessionBoundaryIsArtificialUnresolved(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonSessionBoundary,
	}
	if got := ClassifyContinuity(rt); got != ContinuityArtificialUnresolved {
		t.Errorf("session_boundary: got %s, want %s", got, ContinuityArtificialUnresolved)
	}
}

func TestClassifyContinuity_RejectedLegIsGenuineUnresolved(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonRejectedLeg,
	}
	if got := ClassifyContinuity(rt); got != ContinuityGenuineUnresolved {
		t.Errorf("rejected_leg: got %s, want %s", got, ContinuityGenuineUnresolved)
	}
}

func TestClassifyContinuity_CancelledLegIsGenuineUnresolved(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonCancelledLeg,
	}
	if got := ClassifyContinuity(rt); got != ContinuityGenuineUnresolved {
		t.Errorf("cancelled_leg: got %s, want %s", got, ContinuityGenuineUnresolved)
	}
}

func TestClassifyContinuity_NoExitFoundIsOpen(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonNoExitFound,
	}
	if got := ClassifyContinuity(rt); got != ContinuityOpen {
		t.Errorf("no_exit_found: got %s, want %s", got, ContinuityOpen)
	}
}

func TestClassifyContinuity_QuantityResidueIsOpen(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedEntry,
		UnmatchedReason: ReasonQuantityMismatchResidue,
	}
	if got := ClassifyContinuity(rt); got != ContinuityOpen {
		t.Errorf("quantity_residue: got %s, want %s", got, ContinuityOpen)
	}
}

func TestClassifyContinuity_OrphanExitIsGenuineUnresolved(t *testing.T) {
	rt := RoundTrip{
		State:           StateUnmatchedExit,
		UnmatchedReason: ReasonNoEntryFound,
	}
	if got := ClassifyContinuity(rt); got != ContinuityGenuineUnresolved {
		t.Errorf("orphan_exit: got %s, want %s", got, ContinuityGenuineUnresolved)
	}
}

// --- SessionLeg and IsCrossSession ---

func TestIsCrossSession_DifferentSessions(t *testing.T) {
	a := SessionLeg{SessionID: "session_20260326_100000"}
	b := SessionLeg{SessionID: "session_20260326_140000"}
	if !IsCrossSession(a, b) {
		t.Error("different session IDs should be cross-session")
	}
}

func TestIsCrossSession_SameSession(t *testing.T) {
	a := SessionLeg{SessionID: "session_20260326_100000"}
	b := SessionLeg{SessionID: "session_20260326_100000"}
	if IsCrossSession(a, b) {
		t.Error("same session IDs should not be cross-session")
	}
}

func TestIsCrossSession_EmptySession(t *testing.T) {
	a := SessionLeg{SessionID: ""}
	b := SessionLeg{SessionID: "session_20260326_100000"}
	if IsCrossSession(a, b) {
		t.Error("empty session ID should not be cross-session")
	}
}

// --- CrossSessionWindow validation ---

func TestCrossSessionWindow_Valid(t *testing.T) {
	w := CrossSessionWindow{
		Symbol:    "BTCUSDT",
		Source:    "binance_spot",
		Timeframe: 60,
		Since:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	if !w.Validate() {
		t.Error("valid window should pass validation")
	}
}

func TestCrossSessionWindow_MissingSymbol(t *testing.T) {
	w := CrossSessionWindow{
		Source:    "binance_spot",
		Timeframe: 60,
		Since:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	if w.Validate() {
		t.Error("window without symbol should fail validation")
	}
}

func TestCrossSessionWindow_MissingSource(t *testing.T) {
	w := CrossSessionWindow{
		Symbol:    "BTCUSDT",
		Timeframe: 60,
		Since:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	if w.Validate() {
		t.Error("window without source should fail validation")
	}
}

func TestCrossSessionWindow_ZeroTimeframe(t *testing.T) {
	w := CrossSessionWindow{
		Symbol: "BTCUSDT",
		Source: "binance_spot",
		Since:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	if w.Validate() {
		t.Error("window with zero timeframe should fail validation")
	}
}

func TestCrossSessionWindow_MissingSince(t *testing.T) {
	w := CrossSessionWindow{
		Symbol:    "BTCUSDT",
		Source:    "binance_spot",
		Timeframe: 60,
	}
	if w.Validate() {
		t.Error("window without since should fail validation")
	}
}

// --- CrossSessionLegSet ---

func TestCrossSessionLegSet_ExtractLegs(t *testing.T) {
	legSet := CrossSessionLegSet{
		Legs: []SessionLeg{
			{Leg: makeLeg(LegEntry, execution.SideBuy, "0.1", "5000", t0), SessionID: "s1"},
			{Leg: makeLeg(LegExit, execution.SideSell, "0.1", "5100", t0.Add(time.Hour)), SessionID: "s2"},
		},
	}

	legs := legSet.ExtractLegs()
	if len(legs) != 2 {
		t.Fatalf("ExtractLegs: got %d legs, want 2", len(legs))
	}
	if legs[0].Direction != LegEntry {
		t.Errorf("leg[0].Direction=%s, want entry", legs[0].Direction)
	}
	if legs[1].Direction != LegExit {
		t.Errorf("leg[1].Direction=%s, want exit", legs[1].Direction)
	}
}

func TestCrossSessionLegSet_ExtractLegsEmpty(t *testing.T) {
	legSet := CrossSessionLegSet{}
	legs := legSet.ExtractLegs()
	if legs != nil {
		t.Errorf("empty set should return nil, got %d legs", len(legs))
	}
}

func TestCrossSessionLegSet_SessionLegIndex(t *testing.T) {
	legSet := CrossSessionLegSet{
		Legs: []SessionLeg{
			{Leg: Leg{CorrelationID: "corr-1"}, SessionID: "s1"},
			{Leg: Leg{CorrelationID: "corr-2"}, SessionID: "s2"},
		},
	}

	idx := legSet.SessionLegIndex()
	if len(idx) != 2 {
		t.Fatalf("index size=%d, want 2", len(idx))
	}
	if idx["corr-1"].SessionID != "s1" {
		t.Errorf("corr-1 session=%s, want s1", idx["corr-1"].SessionID)
	}
	if idx["corr-2"].SessionID != "s2" {
		t.Errorf("corr-2 session=%s, want s2", idx["corr-2"].SessionID)
	}
}

func TestCrossSessionLegSet_Counts(t *testing.T) {
	legSet := CrossSessionLegSet{
		Sessions: []string{"s1", "s2"},
		Legs:     make([]SessionLeg, 5),
	}
	if legSet.LegCount() != 5 {
		t.Errorf("LegCount=%d, want 5", legSet.LegCount())
	}
	if legSet.SessionCount() != 2 {
		t.Errorf("SessionCount=%d, want 2", legSet.SessionCount())
	}
}

// --- AnnotateRoundTrips ---

func TestAnnotateRoundTrips_CrossSessionPair(t *testing.T) {
	entryLeg := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000", t0)
	entryLeg.CorrelationID = "corr-entry"
	exitLeg := makeLeg(LegExit, execution.SideSell, "0.1", "5100", t0.Add(time.Hour))
	exitLeg.CorrelationID = "corr-exit"

	roundTrips := []RoundTrip{
		{
			Entry:           &entryLeg,
			Exit:            &exitLeg,
			State:           StatePaired,
			MatchedQuantity: "0.10000000",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
	}

	sessionIndex := map[string]SessionLeg{
		"corr-entry": {Leg: entryLeg, SessionID: "session_20260325_100000"},
		"corr-exit":  {Leg: exitLeg, SessionID: "session_20260326_100000"},
	}

	annotated := AnnotateRoundTrips(roundTrips, sessionIndex)
	if len(annotated) != 1 {
		t.Fatalf("got %d annotated round-trips, want 1", len(annotated))
	}

	csrt := annotated[0]
	if csrt.EntrySessionID != "session_20260325_100000" {
		t.Errorf("entry_session=%s, want session_20260325_100000", csrt.EntrySessionID)
	}
	if csrt.ExitSessionID != "session_20260326_100000" {
		t.Errorf("exit_session=%s, want session_20260326_100000", csrt.ExitSessionID)
	}
	if !csrt.CrossSession {
		t.Error("expected CrossSession=true for different sessions")
	}
	if csrt.Continuity != ContinuityResolved {
		t.Errorf("continuity=%s, want resolved", csrt.Continuity)
	}
}

func TestAnnotateRoundTrips_IntraSessionPair(t *testing.T) {
	entryLeg := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000", t0)
	entryLeg.CorrelationID = "corr-entry"
	exitLeg := makeLeg(LegExit, execution.SideSell, "0.1", "5100", t0.Add(time.Minute))
	exitLeg.CorrelationID = "corr-exit"

	roundTrips := []RoundTrip{
		{
			Entry: &entryLeg, Exit: &exitLeg,
			State: StatePaired, MatchedQuantity: "0.10000000",
			Symbol: "BTCUSDT", Source: "binance_spot",
		},
	}

	sessionIndex := map[string]SessionLeg{
		"corr-entry": {Leg: entryLeg, SessionID: "session_20260326_100000"},
		"corr-exit":  {Leg: exitLeg, SessionID: "session_20260326_100000"},
	}

	annotated := AnnotateRoundTrips(roundTrips, sessionIndex)
	if annotated[0].CrossSession {
		t.Error("expected CrossSession=false for same session")
	}
	if annotated[0].Continuity != ContinuityResolved {
		t.Errorf("continuity=%s, want resolved", annotated[0].Continuity)
	}
}

func TestAnnotateRoundTrips_UnmatchedEntrySessionBoundary(t *testing.T) {
	entryLeg := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000", t0)
	entryLeg.CorrelationID = "corr-entry"

	roundTrips := []RoundTrip{
		{
			Entry:           &entryLeg,
			State:           StateUnmatchedEntry,
			UnmatchedReason: ReasonSessionBoundary,
			MatchedQuantity: "0",
			Symbol:          "BTCUSDT",
			Source:          "binance_spot",
		},
	}

	sessionIndex := map[string]SessionLeg{
		"corr-entry": {Leg: entryLeg, SessionID: "session_20260326_100000"},
	}

	annotated := AnnotateRoundTrips(roundTrips, sessionIndex)
	if annotated[0].Continuity != ContinuityArtificialUnresolved {
		t.Errorf("continuity=%s, want artificial_unresolved", annotated[0].Continuity)
	}
	if annotated[0].CrossSession {
		t.Error("single-leg should not be cross-session")
	}
}

func TestAnnotateRoundTrips_Empty(t *testing.T) {
	annotated := AnnotateRoundTrips(nil, nil)
	if annotated != nil {
		t.Errorf("nil input should return nil, got %d", len(annotated))
	}
}

// --- End-to-end: MatchFIFO on cross-session leg set ---

func TestMatchFIFO_CrossSessionLegsProducePairedRoundTrip(t *testing.T) {
	// Session 1: entry buy at t0
	// Session 2: exit sell at t0 + 4 hours (different session)
	session1Entry := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0)
	session1Entry.CorrelationID = "corr-s1-entry"

	session2Exit := makeLeg(LegExit, execution.SideSell, "0.1", "5200.00", t0.Add(4*time.Hour))
	session2Exit.CorrelationID = "corr-s2-exit"

	legSet := CrossSessionLegSet{
		Sessions: []string{"session_20260326_100000", "session_20260326_140000"},
		Legs: []SessionLeg{
			{Leg: session1Entry, SessionID: "session_20260326_100000"},
			{Leg: session2Exit, SessionID: "session_20260326_140000"},
		},
	}

	// Extract plain legs and run FIFO matching.
	legs := legSet.ExtractLegs()
	roundTrips := MatchFIFO(legs, DefaultMatchingConfig())

	if len(roundTrips) != 1 {
		t.Fatalf("got %d round-trips, want 1", len(roundTrips))
	}
	if roundTrips[0].State != StatePaired {
		t.Errorf("state=%s, want paired", roundTrips[0].State)
	}

	// Annotate with session provenance.
	idx := legSet.SessionLegIndex()
	annotated := AnnotateRoundTrips(roundTrips, idx)
	if !annotated[0].CrossSession {
		t.Error("expected CrossSession=true")
	}
	if annotated[0].Continuity != ContinuityResolved {
		t.Errorf("continuity=%s, want resolved", annotated[0].Continuity)
	}
}

func TestMatchFIFO_CrossSessionPreservesTemporalOrdering(t *testing.T) {
	// Two entries from session 1, one exit from session 2.
	// FIFO should pair the oldest entry first.
	entry1 := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0)
	entry1.CorrelationID = "corr-entry1"

	entry2 := makeLeg(LegEntry, execution.SideBuy, "0.1", "5050.00", t0.Add(30*time.Minute))
	entry2.CorrelationID = "corr-entry2"

	exit1 := makeLeg(LegExit, execution.SideSell, "0.1", "5200.00", t0.Add(4*time.Hour))
	exit1.CorrelationID = "corr-exit1"

	legSet := CrossSessionLegSet{
		Sessions: []string{"s1", "s2"},
		Legs: []SessionLeg{
			{Leg: entry1, SessionID: "s1"},
			{Leg: entry2, SessionID: "s1"},
			{Leg: exit1, SessionID: "s2"},
		},
	}

	legs := legSet.ExtractLegs()
	roundTrips := MatchFIFO(legs, DefaultMatchingConfig())

	var paired, unmatched int
	var pairedEntryCorr string
	for _, rt := range roundTrips {
		switch rt.State {
		case StatePaired:
			paired++
			if rt.Entry != nil {
				pairedEntryCorr = rt.Entry.CorrelationID
			}
		case StateUnmatchedEntry:
			unmatched++
		}
	}

	if paired != 1 {
		t.Errorf("paired=%d, want 1", paired)
	}
	if unmatched != 1 {
		t.Errorf("unmatched=%d, want 1", unmatched)
	}
	// FIFO: oldest entry (corr-entry1) should pair first.
	if pairedEntryCorr != "corr-entry1" {
		t.Errorf("paired entry=%s, want corr-entry1 (FIFO ordering)", pairedEntryCorr)
	}
}

// --- Helper ---

func makeFilledIntent(t *testing.T, side execution.Side, price, qty string) execution.ExecutionIntent {
	t.Helper()
	return execution.ExecutionIntent{
		Type:       "venue_market_order",
		Source:     "binance_spot",
		Instrument: btcUSDTSpot,
		Timeframe:  60,
		Side:       side,
		Quantity:   qty,
		Status:     execution.StatusFilled,
		Fills: []execution.FillRecord{
			{
				Price:     price,
				Quantity:  qty,
				Fee:       "0.50",
				CostBasis: "5000.00",
				Timestamp: t0,
			},
		},
		CorrelationID: "corr-test",
		Timestamp:     t0,
	}
}
