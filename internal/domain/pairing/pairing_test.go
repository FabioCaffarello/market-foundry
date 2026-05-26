package pairing

import (
	"testing"
	"time"

	"internal/domain/execution"
	"internal/domain/instrument"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// --- Type validation tests ---

func TestValidLegDirection(t *testing.T) {
	tests := []struct {
		d     LegDirection
		valid bool
	}{
		{LegEntry, true},
		{LegExit, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidLegDirection(tt.d); got != tt.valid {
			t.Errorf("ValidLegDirection(%q) = %v, want %v", tt.d, got, tt.valid)
		}
	}
}

func TestValidPairingState(t *testing.T) {
	tests := []struct {
		s     PairingState
		valid bool
	}{
		{StatePaired, true},
		{StateUnmatchedEntry, true},
		{StateUnmatchedExit, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidPairingState(tt.s); got != tt.valid {
			t.Errorf("ValidPairingState(%q) = %v, want %v", tt.s, got, tt.valid)
		}
	}
}

// --- IntentToLeg tests ---

func TestIntentToLeg_LongBuyIsEntry(t *testing.T) {
	intent := makeIntent(t, execution.SideBuy, "50000.00", "0.1", "0.50", "5000.00")
	leg := IntentToLeg(intent, "long")
	if leg.Direction != LegEntry {
		t.Errorf("direction=%s, want entry", leg.Direction)
	}
	if leg.Side != execution.SideBuy {
		t.Errorf("side=%s, want buy", leg.Side)
	}
}

func TestIntentToLeg_LongSellIsExit(t *testing.T) {
	intent := makeIntent(t, execution.SideSell, "51000.00", "0.1", "0.50", "5100.00")
	leg := IntentToLeg(intent, "long")
	if leg.Direction != LegExit {
		t.Errorf("direction=%s, want exit", leg.Direction)
	}
}

func TestIntentToLeg_ShortSellIsEntry(t *testing.T) {
	intent := makeIntent(t, execution.SideSell, "50000.00", "0.1", "0.50", "5000.00")
	leg := IntentToLeg(intent, "short")
	if leg.Direction != LegEntry {
		t.Errorf("direction=%s, want entry", leg.Direction)
	}
}

func TestIntentToLeg_ShortBuyIsExit(t *testing.T) {
	intent := makeIntent(t, execution.SideBuy, "49000.00", "0.1", "0.50", "4900.00")
	leg := IntentToLeg(intent, "short")
	if leg.Direction != LegExit {
		t.Errorf("direction=%s, want exit", leg.Direction)
	}
}

func TestIntentToLeg_DefaultDirectionIsLong(t *testing.T) {
	intent := makeIntent(t, execution.SideBuy, "50000.00", "0.1", "0.50", "5000.00")
	leg := IntentToLeg(intent, "")
	if leg.Direction != LegEntry {
		t.Errorf("direction=%s, want entry (default long)", leg.Direction)
	}
}

func TestIntentToLeg_AggregatesMultipleFills(t *testing.T) {
	intent := execution.ExecutionIntent{
		Type:       "market",
		Source:     "binance_spot",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       execution.SideBuy,
		Quantity:   "0.1",
		Status:     execution.StatusFilled,
		Risk:       execution.RiskInput{Type: "ema_crossover", Disposition: "approved"},
		Fills: []execution.FillRecord{
			{Price: "50000.00", Quantity: "0.05", Fee: "0.25", CostBasis: "2500.00", Timestamp: t0},
			{Price: "50100.00", Quantity: "0.05", Fee: "0.25", CostBasis: "2505.00", Timestamp: t0.Add(time.Second)},
		},
		CorrelationID: "corr-multi",
		Timestamp:     t0,
	}
	leg := IntentToLeg(intent, "long")
	if leg.Quantity != formatFloat(0.1) {
		t.Errorf("quantity=%s, want 0.10000000", leg.Quantity)
	}
	if leg.Fee != formatFloat(0.5) {
		t.Errorf("fee=%s, want 0.50000000", leg.Fee)
	}
	if leg.CostBasis != formatFloat(5005.0) {
		t.Errorf("cost_basis=%s, want 5005.00000000", leg.CostBasis)
	}
	if leg.CorrelationID != "corr-multi" {
		t.Errorf("correlation_id=%s, want corr-multi", leg.CorrelationID)
	}
}

func TestIntentToLeg_NoFillsFallback(t *testing.T) {
	intent := execution.ExecutionIntent{
		Type:       "market",
		Source:     "binance_spot",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       execution.SideBuy,
		Quantity:   "0.1",
		Status:     execution.StatusSubmitted,
		Risk:       execution.RiskInput{Type: "ema_crossover", Disposition: "approved"},
		Timestamp:  t0,
	}
	leg := IntentToLeg(intent, "long")
	if leg.Price != "0" {
		t.Errorf("price=%s, want 0", leg.Price)
	}
	if leg.Quantity != "0.1" {
		t.Errorf("quantity=%s, want 0.1", leg.Quantity)
	}
}

// --- FIFO matching tests ---

func TestMatchFIFO_EmptyInput(t *testing.T) {
	rts := MatchFIFO(nil, DefaultMatchingConfig())
	if rts != nil {
		t.Errorf("expected nil, got %d round-trips", len(rts))
	}
}

func TestMatchFIFO_SingleEntryUnmatched(t *testing.T) {
	legs := []Leg{makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0)}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	if len(rts) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(rts))
	}
	if rts[0].State != StateUnmatchedEntry {
		t.Errorf("state=%s, want unmatched_entry", rts[0].State)
	}
	if rts[0].UnmatchedReason != ReasonNoExitFound {
		t.Errorf("reason=%s, want no_exit_found", rts[0].UnmatchedReason)
	}
}

func TestMatchFIFO_SingleExitUnmatched(t *testing.T) {
	legs := []Leg{makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0)}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	if len(rts) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(rts))
	}
	if rts[0].State != StateUnmatchedExit {
		t.Errorf("state=%s, want unmatched_exit", rts[0].State)
	}
	if rts[0].UnmatchedReason != ReasonNoEntryFound {
		t.Errorf("reason=%s, want no_entry_found", rts[0].UnmatchedReason)
	}
}

func TestMatchFIFO_PerfectPair(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	if len(rts) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(rts))
	}
	if rts[0].State != StatePaired {
		t.Errorf("state=%s, want paired", rts[0].State)
	}
	if rts[0].Entry == nil || rts[0].Exit == nil {
		t.Fatal("paired round-trip must have both legs")
	}
	if rts[0].MatchedQuantity != formatFloat(0.1) {
		t.Errorf("matched_quantity=%s, want 0.10000000", rts[0].MatchedQuantity)
	}
	if rts[0].Symbol != "BTCUSDT" {
		t.Errorf("symbol=%s, want BTCUSDT", rts[0].Symbol)
	}
}

func TestMatchFIFO_ShortPair(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideSell, "0.1", "5000.00", t0),
		makeLeg(LegExit, execution.SideBuy, "0.1", "4900.00", t0.Add(time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	if len(rts) != 1 {
		t.Fatalf("expected 1 round-trip, got %d", len(rts))
	}
	if rts[0].State != StatePaired {
		t.Errorf("state=%s, want paired", rts[0].State)
	}
}

func TestMatchFIFO_TemporalOrderingEnforced(t *testing.T) {
	// Exit before entry — should not pair.
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0.Add(time.Hour)),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	// Should produce 2 unmatched.
	if len(rts) != 2 {
		t.Fatalf("expected 2 unmatched, got %d", len(rts))
	}
	for _, rt := range rts {
		if rt.State == StatePaired {
			t.Error("should not pair when exit precedes entry")
		}
	}
}

func TestMatchFIFO_DifferentSymbolsDoNotPair(t *testing.T) {
	entry := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0)
	exit := makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute))
	exit.Symbol = "ETHUSDT"

	rts := MatchFIFO([]Leg{entry, exit}, DefaultMatchingConfig())
	for _, rt := range rts {
		if rt.State == StatePaired {
			t.Error("different symbols should not pair")
		}
	}
}

func TestMatchFIFO_DifferentSourcesDoNotPair(t *testing.T) {
	entry := makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0)
	exit := makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute))
	exit.Source = "binance_futures"

	rts := MatchFIFO([]Leg{entry, exit}, DefaultMatchingConfig())
	for _, rt := range rts {
		if rt.State == StatePaired {
			t.Error("different sources should not pair")
		}
	}
}

func TestMatchFIFO_SameSideDoesNotPair(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0),
		makeLeg(LegExit, execution.SideBuy, "0.1", "5100.00", t0.Add(time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	for _, rt := range rts {
		if rt.State == StatePaired {
			t.Error("same side should not pair")
		}
	}
}

func TestMatchFIFO_PartialMatchProducesRemainder(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.2", "10000.00", t0),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())

	var paired, unmatched int
	for _, rt := range rts {
		switch rt.State {
		case StatePaired:
			paired++
			if rt.MatchedQuantity != formatFloat(0.1) {
				t.Errorf("matched_quantity=%s, want 0.10000000", rt.MatchedQuantity)
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
}

func TestMatchFIFO_MultiplePairsFIFOOrder(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0),
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5050.00", t0.Add(time.Minute)),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(2*time.Minute)),
		makeLeg(LegExit, execution.SideSell, "0.1", "5200.00", t0.Add(3*time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())

	paired := 0
	for _, rt := range rts {
		if rt.State == StatePaired {
			paired++
		}
	}
	if paired != 2 {
		t.Errorf("paired=%d, want 2", paired)
	}
}

func TestMatchFIFO_DeterministicOutput(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0.1", "5000.00", t0),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute)),
		makeLeg(LegEntry, execution.SideBuy, "0.05", "2500.00", t0.Add(2*time.Minute)),
	}

	rts1 := MatchFIFO(legs, DefaultMatchingConfig())
	rts2 := MatchFIFO(legs, DefaultMatchingConfig())

	if len(rts1) != len(rts2) {
		t.Fatalf("non-deterministic: run1=%d, run2=%d", len(rts1), len(rts2))
	}
	for i := range rts1 {
		if rts1[i].State != rts2[i].State {
			t.Errorf("non-deterministic at %d: state1=%s, state2=%s", i, rts1[i].State, rts2[i].State)
		}
	}
}

func TestMatchFIFO_ZeroQuantitySkipped(t *testing.T) {
	legs := []Leg{
		makeLeg(LegEntry, execution.SideBuy, "0", "0", t0),
		makeLeg(LegExit, execution.SideSell, "0.1", "5100.00", t0.Add(time.Minute)),
	}
	rts := MatchFIFO(legs, DefaultMatchingConfig())
	for _, rt := range rts {
		if rt.State == StatePaired {
			t.Error("zero-quantity leg should not pair")
		}
	}
}

// --- Summarize tests ---

func TestSummarize_Empty(t *testing.T) {
	s := Summarize(nil)
	if s.PairedCount != 0 || s.ResolvedRate != 0 {
		t.Errorf("empty summary should be zero")
	}
}

func TestSummarize_MixedResults(t *testing.T) {
	rts := []RoundTrip{
		{State: StatePaired},
		{State: StatePaired},
		{State: StateUnmatchedEntry},
		{State: StateUnmatchedExit},
	}
	s := Summarize(rts)
	if s.PairedCount != 2 {
		t.Errorf("paired=%d, want 2", s.PairedCount)
	}
	if s.UnmatchedEntries != 1 {
		t.Errorf("unmatched_entries=%d, want 1", s.UnmatchedEntries)
	}
	if s.UnmatchedExits != 1 {
		t.Errorf("unmatched_exits=%d, want 1", s.UnmatchedExits)
	}
	if s.TotalEntries != 3 {
		t.Errorf("total_entries=%d, want 3", s.TotalEntries)
	}
	if s.TotalExits != 3 {
		t.Errorf("total_exits=%d, want 3", s.TotalExits)
	}
	// resolved_rate = 2 / (2+1+1) = 0.5
	if s.ResolvedRate != 0.5 {
		t.Errorf("resolved_rate=%f, want 0.5", s.ResolvedRate)
	}
}

// --- RoundTrip method tests ---

func TestRoundTrip_IsPaired(t *testing.T) {
	rt := RoundTrip{State: StatePaired}
	if !rt.IsPaired() {
		t.Error("expected IsPaired() = true")
	}
	rt.State = StateUnmatchedEntry
	if rt.IsPaired() {
		t.Error("expected IsPaired() = false")
	}
}

func TestRoundTrip_IsOpen(t *testing.T) {
	rt := RoundTrip{State: StateUnmatchedEntry}
	if !rt.IsOpen() {
		t.Error("expected IsOpen() = true")
	}
	rt.State = StatePaired
	if rt.IsOpen() {
		t.Error("expected IsOpen() = false")
	}
}

// --- helpers ---

var t0 = time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)

func makeIntent(t *testing.T, side execution.Side, price, qty, fee, costBasis string) execution.ExecutionIntent {
	t.Helper()
	return execution.ExecutionIntent{
		Type:       "market",
		Source:     "binance_spot",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       side,
		Quantity:   qty,
		Status:     execution.StatusFilled,
		Risk:       execution.RiskInput{Type: "ema_crossover", Disposition: "approved"},
		Fills: []execution.FillRecord{
			{Price: price, Quantity: qty, Fee: fee, CostBasis: costBasis, Timestamp: t0},
		},
		CorrelationID: "corr-test",
		Timestamp:     t0,
	}
}

func makeLeg(dir LegDirection, side execution.Side, qty, costBasis string, ts time.Time) Leg {
	return Leg{
		Direction: dir,
		Side:      side,
		Symbol:    "BTCUSDT",
		Source:    "binance_spot",
		Timeframe: 60,
		Quantity:  qty,
		Price:     "50000.00",
		Fee:       "0.50",
		CostBasis: costBasis,
		Timestamp: ts,
	}
}
