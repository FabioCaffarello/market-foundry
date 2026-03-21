package execution_test

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/executionclient"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
)

// TestPipeline_EvaluateSimulateEmit validates the full derive-side pipeline:
//   risk primitives → PaperOrderEvaluator → PaperFillSimulator → PaperOrderSubmittedEvent
// This is the exact sequence executed by PaperOrderEvaluatorActor.onRiskAssessed.
func TestPipeline_EvaluateSimulateEmit_BuyOrder(t *testing.T) {
	const (
		source    = "binancef"
		symbol    = "btcusdt"
		timeframe = 60
		corrID    = "corr-abc123"
		causeID   = "cause-def456"
	)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	// Step 1: Evaluate risk → execution intent.
	eval := appexec.NewPaperOrderEvaluator(source, symbol, timeframe)
	intent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		timeframe, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}

	// Step 2: Set causal trace (done by actor, not evaluator — evaluator is pure).
	intent.CorrelationID = corrID
	intent.CausationID = causeID

	// Step 3: Simulate paper fill.
	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("fill simulation should succeed")
	}

	// Step 4: Validate intent (actor does this before publishing).
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("intent should be valid after pipeline: %s", prob.Message)
	}

	// Step 5: Construct event (mirrors actor code).
	event := domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(corrID).
			WithCausationID(causeID),
		ExecutionIntent: intent,
	}

	// ---------- Assertions ----------

	// Event metadata.
	if event.Metadata.CorrelationID != corrID {
		t.Fatalf("expected correlation_id %q, got %q", corrID, event.Metadata.CorrelationID)
	}
	if event.Metadata.CausationID != causeID {
		t.Fatalf("expected causation_id %q, got %q", causeID, event.Metadata.CausationID)
	}
	if event.Metadata.ID == "" {
		t.Fatal("event metadata ID should be populated")
	}
	if event.EventName() != domainexec.EventPaperOrderSubmitted {
		t.Fatalf("expected event name %q, got %q", domainexec.EventPaperOrderSubmitted, event.EventName())
	}

	// Intent identity.
	ei := event.ExecutionIntent
	if ei.Type != "paper_order" {
		t.Fatalf("expected type paper_order, got %q", ei.Type)
	}
	if ei.Source != source {
		t.Fatalf("expected source %q, got %q", source, ei.Source)
	}
	if ei.Symbol != symbol {
		t.Fatalf("expected symbol %q, got %q", symbol, ei.Symbol)
	}
	if ei.Timeframe != timeframe {
		t.Fatalf("expected timeframe %d, got %d", timeframe, ei.Timeframe)
	}

	// Execution outcome.
	if ei.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy, got %q", ei.Side)
	}
	if ei.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled after simulation, got %q", ei.Status)
	}
	if ei.Quantity != "0.02" {
		t.Fatalf("expected quantity 0.02, got %q", ei.Quantity)
	}
	if ei.FilledQuantity != "0.02" {
		t.Fatalf("expected filled_quantity 0.02, got %q", ei.FilledQuantity)
	}

	// Fill record.
	if len(ei.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(ei.Fills))
	}
	if !ei.Fills[0].Simulated {
		t.Fatal("expected fill to be simulated")
	}
	if ei.Fills[0].Quantity != "0.02" {
		t.Fatalf("expected fill quantity 0.02, got %q", ei.Fills[0].Quantity)
	}

	// Trace fields persisted in intent.
	if ei.CorrelationID != corrID {
		t.Fatalf("expected intent correlation_id %q, got %q", corrID, ei.CorrelationID)
	}
	if ei.CausationID != causeID {
		t.Fatalf("expected intent causation_id %q, got %q", causeID, ei.CausationID)
	}

	// Risk input.
	if ei.Risk.Type != "position_exposure" {
		t.Fatalf("expected risk.type position_exposure, got %q", ei.Risk.Type)
	}
	if ei.Risk.Disposition != "approved" {
		t.Fatalf("expected risk.disposition approved, got %q", ei.Risk.Disposition)
	}

	// Final flag.
	if !ei.Final {
		t.Fatal("expected Final=true")
	}

	// Partition and dedup keys.
	if ei.PartitionKey() != "binancef.btcusdt.60" {
		t.Fatalf("unexpected partition key: %q", ei.PartitionKey())
	}
	if ei.DeduplicationKey() == "" {
		t.Fatal("dedup key should not be empty")
	}
}

func TestPipeline_EvaluateSimulateEmit_RejectedRisk_NoFill(t *testing.T) {
	ts := time.Now().UTC()

	eval := appexec.NewPaperOrderEvaluator("binancef", "ethusdt", 300)
	intent, ok := eval.Evaluate(
		"position_exposure", "rejected", "0.30", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		300, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed even for rejected risk")
	}

	intent.CorrelationID = "corr-rejected"
	intent.CausationID = "cause-rejected"

	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("simulation should succeed for no-action intent")
	}

	if prob := intent.Validate(); prob != nil {
		t.Fatalf("rejected intent should be valid: %s", prob.Message)
	}

	// No-action intent: stays submitted, no fills.
	if intent.Side != domainexec.SideNone {
		t.Fatalf("expected SideNone, got %q", intent.Side)
	}
	if intent.Status != domainexec.StatusSubmitted {
		t.Fatalf("expected StatusSubmitted, got %q", intent.Status)
	}
	if len(intent.Fills) != 0 {
		t.Fatalf("expected 0 fills, got %d", len(intent.Fills))
	}

	// Trace still preserved.
	if intent.CorrelationID != "corr-rejected" {
		t.Fatalf("trace lost: expected corr-rejected, got %q", intent.CorrelationID)
	}
}

// TestPipeline_MultiSymbol_FullIsolation validates that the complete pipeline
// produces independent, non-colliding results across multiple symbols.
func TestPipeline_MultiSymbol_FullIsolation(t *testing.T) {
	type symbolCase struct {
		symbol    string
		direction string
		wantSide  domainexec.Side
		wantFills int
	}
	cases := []symbolCase{
		{"btcusdt", "long", domainexec.SideBuy, 1},
		{"ethusdt", "short", domainexec.SideSell, 1},
		{"solusdt", "flat", domainexec.SideNone, 0},
	}
	timeframes := []int{60, 300}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	partitionKeys := make(map[string]string)
	dedupKeys := make(map[string]string)

	for _, tc := range cases {
		for _, tf := range timeframes {
			eval := appexec.NewPaperOrderEvaluator("binancef", tc.symbol, tf)
			intent, ok := eval.Evaluate(
				"position_exposure", "approved", "0.85", "0.02",
				tc.direction, "0.72",
				"mean_reversion_entry", "high",
				tf, ts,
			)
			if !ok {
				t.Fatalf("%s/%d: evaluation failed", tc.symbol, tf)
			}

			intent.CorrelationID = "corr-" + tc.symbol
			intent.CausationID = "cause-" + tc.symbol

			sim := &appexec.PaperFillSimulator{}
			intent, ok = sim.SimulateFill(intent)
			if !ok {
				t.Fatalf("%s/%d: simulation failed", tc.symbol, tf)
			}

			if prob := intent.Validate(); prob != nil {
				t.Fatalf("%s/%d: invalid: %s", tc.symbol, tf, prob.Message)
			}

			// Verify side.
			if intent.Side != tc.wantSide {
				t.Fatalf("%s/%d: expected side %q, got %q", tc.symbol, tf, tc.wantSide, intent.Side)
			}

			// Verify fills count.
			if len(intent.Fills) != tc.wantFills {
				t.Fatalf("%s/%d: expected %d fills, got %d", tc.symbol, tf, tc.wantFills, len(intent.Fills))
			}

			// Verify symbol ownership.
			if intent.Symbol != tc.symbol {
				t.Fatalf("%s/%d: symbol bleed: got %q", tc.symbol, tf, intent.Symbol)
			}

			// Verify trace ownership.
			if intent.CorrelationID != "corr-"+tc.symbol {
				t.Fatalf("%s/%d: correlation bleed: got %q", tc.symbol, tf, intent.CorrelationID)
			}

			// Check partition key isolation.
			pk := intent.PartitionKey()
			if existing, collision := partitionKeys[pk]; collision {
				t.Fatalf("partition key collision: %q used by %q and %q", pk, existing, tc.symbol)
			}
			partitionKeys[pk] = tc.symbol

			// Check dedup key isolation.
			dk := intent.DeduplicationKey()
			if existing, collision := dedupKeys[dk]; collision {
				t.Fatalf("dedup key collision: %q used by %q and %q", dk, existing, tc.symbol)
			}
			dedupKeys[dk] = tc.symbol
		}
	}

	expectedKeys := len(cases) * len(timeframes)
	if len(partitionKeys) != expectedKeys {
		t.Fatalf("expected %d unique partition keys, got %d", expectedKeys, len(partitionKeys))
	}
	if len(dedupKeys) != expectedKeys {
		t.Fatalf("expected %d unique dedup keys, got %d", expectedKeys, len(dedupKeys))
	}
}

// ---------- S84: Integrated venue + fill chain tests ----------

// TestPipeline_VenueAdapter_FullChain_DeriveToFill validates the complete chain:
//
//	evaluate → simulate → venue submit → fill event construction with trace preservation.
//
// This proves the derive → execute → store event chain at the application layer.
func TestPipeline_VenueAdapter_FullChain_DeriveToFill(t *testing.T) {
	const (
		source    = "binancef"
		symbol    = "btcusdt"
		timeframe = 60
		corrID    = "corr-chain-001"
		causeID   = "cause-chain-001"
	)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	// Step 1: Derive-side — evaluate risk → execution intent.
	eval := appexec.NewPaperOrderEvaluator(source, symbol, timeframe)
	intent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		timeframe, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}

	intent.CorrelationID = corrID
	intent.CausationID = causeID

	// Step 2: Derive-side — simulate paper fill.
	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("fill simulation should succeed")
	}

	// Step 3: Construct PaperOrderSubmittedEvent (as derive publisher would).
	submitEvent := domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(corrID).
			WithCausationID(causeID),
		ExecutionIntent: intent,
	}

	// Step 4: Execute-side — venue adapter submits order via PaperVenueAdapter.
	venue := appexec.NewPaperVenueAdapter(0)
	receipt, prob := venue.SubmitOrder(nil, ports.VenueOrderRequest{Intent: submitEvent.ExecutionIntent})
	if prob != nil {
		t.Fatalf("venue submit failed: %s", prob.Message)
	}

	// Step 5: Execute-side — construct VenueOrderFilledEvent (as venue adapter actor would).
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(submitEvent.Metadata.CorrelationID).
			WithCausationID(submitEvent.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// ---------- Assertions: trace chain ----------

	// Correlation ID must flow from derive to execute.
	if fillEvent.Metadata.CorrelationID != corrID {
		t.Fatalf("trace broken: expected correlation_id %q in fill event, got %q", corrID, fillEvent.Metadata.CorrelationID)
	}

	// Causation ID in fill event must be the submit event's ID (causal link).
	if fillEvent.Metadata.CausationID != submitEvent.Metadata.ID {
		t.Fatalf("causation chain broken: expected fill causation_id=%q (submit event ID), got %q", submitEvent.Metadata.ID, fillEvent.Metadata.CausationID)
	}

	// Fill event must have its own unique ID.
	if fillEvent.Metadata.ID == "" {
		t.Fatal("fill event should have its own metadata ID")
	}
	if fillEvent.Metadata.ID == submitEvent.Metadata.ID {
		t.Fatal("fill event ID should differ from submit event ID")
	}

	// ---------- Assertions: venue order ----------

	if receipt.VenueOrderID == "" {
		t.Fatal("venue order ID should be generated")
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected venue status filled, got %q", receipt.Status)
	}

	// ---------- Assertions: fill event intent ----------

	ei := fillEvent.ExecutionIntent
	if ei.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled status in fill event, got %q", ei.Status)
	}
	if ei.Side != domainexec.SideBuy {
		t.Fatalf("expected buy side, got %q", ei.Side)
	}
	if len(ei.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(ei.Fills))
	}
	if !ei.Fills[0].Simulated {
		t.Fatal("paper fill must be simulated")
	}
	if ei.Symbol != symbol {
		t.Fatalf("symbol bleed: expected %q, got %q", symbol, ei.Symbol)
	}

	// Venue order ID carried through.
	if fillEvent.VenueOrderID == "" {
		t.Fatal("venue order ID should be present in fill event")
	}

	// ---------- Assertions: intent validation ----------

	if prob := ei.Validate(); prob != nil {
		t.Fatalf("fill event intent should be valid: %s", prob.Message)
	}
}

// TestPipeline_VenueAdapter_NoAction_NoFillRecord validates that no-action intents
// pass through the venue adapter without generating fill records.
func TestPipeline_VenueAdapter_NoAction_NoFillRecord(t *testing.T) {
	ts := time.Now().UTC()

	eval := appexec.NewPaperOrderEvaluator("binancef", "ethusdt", 300)
	intent, ok := eval.Evaluate(
		"position_exposure", "rejected", "0.30", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		300, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed for rejected risk")
	}

	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("simulation should succeed")
	}

	// Submit to venue adapter.
	venue := appexec.NewPaperVenueAdapter(0)
	receipt, prob := venue.SubmitOrder(nil, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("venue submit failed: %s", prob.Message)
	}

	// No-action: accepted, no new fills.
	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("expected accepted for no-action, got %q", receipt.Status)
	}
	if receipt.Intent.Side != domainexec.SideNone {
		t.Fatalf("expected SideNone, got %q", receipt.Intent.Side)
	}
}

// TestPipeline_StalenessGuard_Integration validates that stale intents are correctly
// detected and would be blocked by the execution pipeline.
func TestPipeline_StalenessGuard_Integration(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// Simulate a fresh intent from derive.
	eval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	freshIntent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, now.Add(-30*time.Second),
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}

	if guard.IsStale(freshIntent.Timestamp, now) {
		t.Fatal("30s-old intent should not be stale")
	}

	// Simulate a stale intent.
	staleIntent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, now.Add(-5*time.Minute),
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}

	if !guard.IsStale(staleIntent.Timestamp, now) {
		t.Fatal("5min-old intent should be stale with 2min guard")
	}

	// Confirm that a fresh intent passes through venue adapter.
	sim := &appexec.PaperFillSimulator{}
	freshIntent, _ = sim.SimulateFill(freshIntent)
	venue := appexec.NewPaperVenueAdapter(0)
	receipt, prob := venue.SubmitOrder(nil, ports.VenueOrderRequest{Intent: freshIntent})
	if prob != nil {
		t.Fatalf("fresh intent should pass venue: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %q", receipt.Status)
	}
}

// TestPipeline_StatusPropagation_IntentAndResult validates the DeriveEffectivePropagation
// logic that powers the composite /execution/status/latest endpoint.
func TestPipeline_StatusPropagation_IntentAndResult(t *testing.T) {
	ts := time.Now().UTC()

	cases := []struct {
		name       string
		intent     *domainexec.ExecutionIntent
		result     *domainexec.ExecutionIntent
		wantProp   string
	}{
		{
			name:     "both nil → none",
			intent:   nil,
			result:   nil,
			wantProp: "none",
		},
		{
			name: "intent only (submitted) → submitted",
			intent: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideNone, Quantity: "0",
				Status: domainexec.StatusSubmitted, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "rejected", Confidence: "0.3", Timeframe: 60},
			},
			result:   nil,
			wantProp: "submitted",
		},
		{
			name: "intent (submitted) + result (filled) → filled (result wins)",
			intent: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.02",
				Status: domainexec.StatusFilled, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.02",
				FilledQuantity: "0.02", Status: domainexec.StatusFilled, Final: true, Timestamp: ts,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "0", Quantity: "0.02", Fee: "0", Simulated: true, Timestamp: ts}},
			},
			wantProp: "filled",
		},
		{
			name:   "result only (filled) → filled",
			intent: nil,
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.02",
				FilledQuantity: "0.02", Status: domainexec.StatusFilled, Final: true, Timestamp: ts,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "0", Quantity: "0.02", Fee: "0", Simulated: true, Timestamp: ts}},
			},
			wantProp: "filled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := executionclient.DeriveEffectivePropagation(tc.intent, tc.result)
			if got != tc.wantProp {
				t.Fatalf("expected propagation %q, got %q", tc.wantProp, got)
			}
		})
	}
}

// TestPipeline_MultiSymbol_FillIsolation validates that the full chain from evaluate through
// venue adapter maintains strict per-symbol isolation for fill events.
func TestPipeline_MultiSymbol_FillIsolation(t *testing.T) {
	type symbolCase struct {
		symbol    string
		direction string
		wantSide  domainexec.Side
	}
	cases := []symbolCase{
		{"btcusdt", "long", domainexec.SideBuy},
		{"ethusdt", "short", domainexec.SideSell},
		{"solusdt", "flat", domainexec.SideNone},
	}
	timeframes := []int{60, 300}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	venue := appexec.NewPaperVenueAdapter(0)

	venueOrderIDs := make(map[string]bool)

	for _, tc := range cases {
		for _, tf := range timeframes {
			eval := appexec.NewPaperOrderEvaluator("binancef", tc.symbol, tf)
			intent, ok := eval.Evaluate(
				"position_exposure", "approved", "0.85", "0.02",
				tc.direction, "0.72",
				"mean_reversion_entry", "high",
				tf, ts,
			)
			if !ok {
				t.Fatalf("%s/%d: evaluation failed", tc.symbol, tf)
			}

			intent.CorrelationID = "corr-" + tc.symbol
			intent.CausationID = "cause-" + tc.symbol

			sim := &appexec.PaperFillSimulator{}
			intent, ok = sim.SimulateFill(intent)
			if !ok {
				t.Fatalf("%s/%d: simulation failed", tc.symbol, tf)
			}

			// Submit to venue.
			receipt, prob := venue.SubmitOrder(nil, ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("%s/%d: venue submit failed: %s", tc.symbol, tf, prob.Message)
			}

			// Verify symbol ownership preserved through venue.
			if receipt.Intent.Symbol != tc.symbol {
				t.Fatalf("%s/%d: symbol bleed in receipt: got %q", tc.symbol, tf, receipt.Intent.Symbol)
			}

			// Verify side preserved.
			if receipt.Intent.Side != tc.wantSide {
				t.Fatalf("%s/%d: side mismatch: expected %q, got %q", tc.symbol, tf, tc.wantSide, receipt.Intent.Side)
			}

			// Verify trace ownership preserved.
			if receipt.Intent.CorrelationID != "corr-"+tc.symbol {
				t.Fatalf("%s/%d: correlation bleed: got %q", tc.symbol, tf, receipt.Intent.CorrelationID)
			}

			// Verify venue order IDs are unique across all symbols.
			if venueOrderIDs[receipt.VenueOrderID] {
				t.Fatalf("%s/%d: duplicate venue order ID: %q", tc.symbol, tf, receipt.VenueOrderID)
			}
			venueOrderIDs[receipt.VenueOrderID] = true
		}
	}

	// Verify expected number of unique venue order IDs.
	expectedCount := len(cases) * len(timeframes)
	if len(venueOrderIDs) != expectedCount {
		t.Fatalf("expected %d unique venue order IDs, got %d", expectedCount, len(venueOrderIDs))
	}
}
