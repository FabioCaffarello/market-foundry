package execution_test

import (
	"fmt"
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

func instrumentForBase(t *testing.T, base string, contract instrument.ContractType) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New(base, "USDT", contract)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func validIntent(t *testing.T) execution.ExecutionIntent {
	t.Helper()
	return execution.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       execution.SideBuy,
		Quantity:   "0.02",
		Status:     execution.StatusSubmitted,
		Risk: execution.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Parameters: map[string]string{"max_position_pct": "0.02"},
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}
}

// ---------- Validation ----------

func TestExecutionIntent_Validate_Valid(t *testing.T) {
	ei := validIntent(t)
	if prob := ei.Validate(); prob != nil {
		t.Fatalf("expected valid, got: %s", prob.Message)
	}
}

func TestExecutionIntent_Validate_EmptyType(t *testing.T) {
	ei := validIntent(t)
	ei.Type = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestExecutionIntent_Validate_EmptySource(t *testing.T) {
	ei := validIntent(t)
	ei.Source = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty source")
	}
}

func TestExecutionIntent_Validate_EmptySymbol(t *testing.T) {
	ei := validIntent(t)
	ei.Instrument = instrument.CanonicalInstrument{}
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty symbol")
	}
}

func TestExecutionIntent_Validate_ZeroTimeframe(t *testing.T) {
	ei := validIntent(t)
	ei.Timeframe = 0
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timeframe")
	}
}

func TestExecutionIntent_Validate_InvalidSide(t *testing.T) {
	ei := validIntent(t)
	ei.Side = "invalid"
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid side")
	}
}

func TestExecutionIntent_Validate_EmptySide(t *testing.T) {
	ei := validIntent(t)
	ei.Side = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty side")
	}
}

func TestExecutionIntent_Validate_InvalidStatus(t *testing.T) {
	ei := validIntent(t)
	ei.Status = "invalid"
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestExecutionIntent_Validate_AllStatuses(t *testing.T) {
	statuses := []execution.Status{
		execution.StatusSubmitted,
		execution.StatusAccepted,
		execution.StatusFilled,
		execution.StatusPartiallyFilled,
		execution.StatusRejected,
		execution.StatusCancelled,
	}
	for _, st := range statuses {
		ei := validIntent(t)
		ei.Status = st
		if prob := ei.Validate(); prob != nil {
			t.Fatalf("status %q should be valid, got: %s", st, prob.Message)
		}
	}
}

func TestExecutionIntent_Validate_EmptyStatus(t *testing.T) {
	ei := validIntent(t)
	ei.Status = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty status")
	}
}

func TestExecutionIntent_Validate_EmptyQuantity(t *testing.T) {
	ei := validIntent(t)
	ei.Quantity = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty quantity")
	}
}

func TestExecutionIntent_Validate_EmptyRiskType(t *testing.T) {
	ei := validIntent(t)
	ei.Risk.Type = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty risk.type")
	}
}

func TestExecutionIntent_Validate_EmptyRiskDisposition(t *testing.T) {
	ei := validIntent(t)
	ei.Risk.Disposition = ""
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for empty risk.disposition")
	}
}

func TestExecutionIntent_Validate_ZeroTimestamp(t *testing.T) {
	ei := validIntent(t)
	ei.Timestamp = time.Time{}
	if prob := ei.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timestamp")
	}
}

func TestExecutionIntent_Validate_AllSides(t *testing.T) {
	for _, side := range []execution.Side{execution.SideBuy, execution.SideSell, execution.SideNone} {
		ei := validIntent(t)
		ei.Side = side
		if prob := ei.Validate(); prob != nil {
			t.Fatalf("side %q should be valid, got: %s", side, prob.Message)
		}
	}
}

// ---------- Partition Key ----------

func TestExecutionIntent_PartitionKey(t *testing.T) {
	ei := execution.ExecutionIntent{Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60}
	expected := "binancef.btc_usdt_perpetual.60"
	if got := ei.PartitionKey(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// ---------- Deduplication Key ----------

func TestExecutionIntent_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	ei := execution.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Timestamp:  ts,
	}
	got := ei.DeduplicationKey()
	prefix := "exec:paper_order:binancef:btcusdt:60:"
	if got[:len(prefix)] != prefix {
		t.Fatalf("expected prefix %q, got %q", prefix, got)
	}
	// P4.1.11.a: dedup key precision raised from Unix() to UnixNano()
	// across ExecutionIntent/Decision/Risk/Signal to complete the
	// P4.1.10 Strategy fix (same root cause, same recipe).
	expectedSuffix := fmt.Sprintf("%d", ts.UnixNano())
	if got[len(prefix):] != expectedSuffix {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(prefix):])
	}
}

// ---------- Multi-Symbol Isolation ----------

func TestExecutionIntent_MultiSymbol_PartitionKeyIsolation(t *testing.T) {
	bases := []string{"BTC", "ETH", "SOL"}
	timeframes := []int{60, 300}
	keys := make(map[string]string) // partition key → base

	for _, base := range bases {
		for _, tf := range timeframes {
			ei := execution.ExecutionIntent{Source: "binancef", Instrument: instrumentForBase(t, base, instrument.ContractPerpetual), Timeframe: tf}
			key := ei.PartitionKey()
			if existing, collision := keys[key]; collision {
				t.Fatalf("partition key collision: %q used by both %q and %q", key, existing, base)
			}
			keys[key] = base
		}
	}

	expectedCount := len(bases) * len(timeframes)
	if len(keys) != expectedCount {
		t.Fatalf("expected %d unique partition keys, got %d", expectedCount, len(keys))
	}
}

func TestExecutionIntent_MultiSymbol_DeduplicationKeyIsolation(t *testing.T) {
	bases := []string{"BTC", "ETH"}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	dedupKeys := make(map[string]string)

	for _, base := range bases {
		ei := execution.ExecutionIntent{
			Type:       "paper_order",
			Source:     "binancef",
			Instrument: instrumentForBase(t, base, instrument.ContractPerpetual),
			Timeframe:  60,
			Timestamp:  ts,
		}
		key := ei.DeduplicationKey()
		if existing, collision := dedupKeys[key]; collision {
			t.Fatalf("dedup key collision: %q used by both %q and %q", key, existing, base)
		}
		dedupKeys[key] = base
	}

	if len(dedupKeys) != len(bases) {
		t.Fatalf("expected %d unique dedup keys, got %d", len(bases), len(dedupKeys))
	}
}

func TestExecutionIntent_MultiSymbol_NoOwnershipBleed(t *testing.T) {
	e1 := validIntent(t)
	e1.Instrument = btcUSDTPerp(t)

	e2 := validIntent(t)
	e2.Instrument = instrumentForBase(t, "ETH", instrument.ContractPerpetual)

	if e1.VenueSymbol() == e2.VenueSymbol() {
		t.Fatal("symbols should differ")
	}
	if e1.PartitionKey() == e2.PartitionKey() {
		t.Fatalf("partition keys should differ: %q vs %q", e1.PartitionKey(), e2.PartitionKey())
	}
	if e1.Source != e2.Source {
		t.Fatal("source should be shared across symbols")
	}
	if e1.Type != e2.Type {
		t.Fatal("type should be shared across symbols")
	}
	// Validate both independently pass validation.
	if prob := e1.Validate(); prob != nil {
		t.Fatalf("e1 should be valid: %s", prob.Message)
	}
	if prob := e2.Validate(); prob != nil {
		t.Fatalf("e2 should be valid: %s", prob.Message)
	}
}

// ---------- Lifecycle Transitions ----------

func TestValidTransition_SubmittedToAccepted(t *testing.T) {
	if !execution.ValidTransition(execution.StatusSubmitted, execution.StatusAccepted) {
		t.Fatal("submitted → accepted should be valid")
	}
}

func TestValidTransition_SubmittedToRejected(t *testing.T) {
	if !execution.ValidTransition(execution.StatusSubmitted, execution.StatusRejected) {
		t.Fatal("submitted → rejected should be valid")
	}
}

func TestValidTransition_AcceptedToFilled(t *testing.T) {
	if !execution.ValidTransition(execution.StatusAccepted, execution.StatusFilled) {
		t.Fatal("accepted → filled should be valid")
	}
}

func TestValidTransition_AcceptedToPartiallyFilled(t *testing.T) {
	if !execution.ValidTransition(execution.StatusAccepted, execution.StatusPartiallyFilled) {
		t.Fatal("accepted → partially_filled should be valid")
	}
}

func TestValidTransition_AcceptedToCancelled(t *testing.T) {
	if !execution.ValidTransition(execution.StatusAccepted, execution.StatusCancelled) {
		t.Fatal("accepted → cancelled should be valid")
	}
}

func TestValidTransition_PartiallyFilledToFilled(t *testing.T) {
	if !execution.ValidTransition(execution.StatusPartiallyFilled, execution.StatusFilled) {
		t.Fatal("partially_filled → filled should be valid")
	}
}

func TestValidTransition_PartiallyFilledToCancelled(t *testing.T) {
	if !execution.ValidTransition(execution.StatusPartiallyFilled, execution.StatusCancelled) {
		t.Fatal("partially_filled → cancelled should be valid")
	}
}

func TestValidTransition_TerminalStatesCannotTransition(t *testing.T) {
	terminals := []execution.Status{execution.StatusFilled, execution.StatusRejected, execution.StatusCancelled}
	allStatuses := []execution.Status{
		execution.StatusSubmitted, execution.StatusAccepted, execution.StatusFilled,
		execution.StatusPartiallyFilled, execution.StatusRejected, execution.StatusCancelled,
	}
	for _, from := range terminals {
		for _, to := range allStatuses {
			if execution.ValidTransition(from, to) {
				t.Fatalf("terminal %q → %q should be invalid", from, to)
			}
		}
	}
}

func TestValidTransition_InvalidTransitions(t *testing.T) {
	invalid := [][2]execution.Status{
		{execution.StatusSubmitted, execution.StatusFilled},
		{execution.StatusSubmitted, execution.StatusPartiallyFilled},
		{execution.StatusSubmitted, execution.StatusCancelled},
		{execution.StatusAccepted, execution.StatusSubmitted},
		{execution.StatusAccepted, execution.StatusRejected},
	}
	for _, pair := range invalid {
		if execution.ValidTransition(pair[0], pair[1]) {
			t.Fatalf("%q → %q should be invalid", pair[0], pair[1])
		}
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	if !execution.StatusFilled.IsTerminal() {
		t.Fatal("filled should be terminal")
	}
	if !execution.StatusRejected.IsTerminal() {
		t.Fatal("rejected should be terminal")
	}
	if !execution.StatusCancelled.IsTerminal() {
		t.Fatal("cancelled should be terminal")
	}
	if execution.StatusSubmitted.IsTerminal() {
		t.Fatal("submitted should not be terminal")
	}
	if execution.StatusAccepted.IsTerminal() {
		t.Fatal("accepted should not be terminal")
	}
	if execution.StatusPartiallyFilled.IsTerminal() {
		t.Fatal("partially_filled should not be terminal")
	}
}

// ---------- Fill Records ----------

func TestFillRecord_FilledIntentValidation(t *testing.T) {
	ei := validIntent(t)
	ei.Status = execution.StatusFilled
	ei.FilledQuantity = "0.02"
	ei.Fills = []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.02", Fee: "0.01", Simulated: true, Timestamp: ei.Timestamp},
	}
	if prob := ei.Validate(); prob != nil {
		t.Fatalf("filled intent should be valid: %s", prob.Message)
	}
}

func TestExecutionIntent_MultiSymbol_CrossTimeframe_NoCollision(t *testing.T) {
	// Same symbol, different timeframes → must produce distinct partition keys.
	inst := btcUSDTPerp(t)
	timeframes := []int{60, 300, 900}
	keys := make(map[string]bool)

	for _, tf := range timeframes {
		ei := execution.ExecutionIntent{Source: "binancef", Instrument: inst, Timeframe: tf}
		key := ei.PartitionKey()
		if keys[key] {
			t.Fatalf("partition key collision for same symbol different timeframe: %q", key)
		}
		keys[key] = true
	}
}
