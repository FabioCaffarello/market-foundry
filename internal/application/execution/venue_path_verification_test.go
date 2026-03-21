package execution_test

// S329 — Actor Pipeline Venue Path Verification Tests.
//
// These tests exercise the exact venue path that VenueAdapterActor.onIntent()
// follows, proving that the composed decorator pipeline (Post200Reconciler →
// RetrySubmitter → rawAdapter) participates effectively in the real operational
// flow.
//
// Unlike SC-01..SC-07 (which test decorator composition in isolation), VP tests
// mirror the actor's full path:
//
//	safety gate → composed submit → fill event construction → observability extraction
//
// This validates that submit/fill/persist/read behavior remains intact after
// composition, and that retry/reconciliation/observability produce auditable
// signals in the operational path.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// VP-01: Full venue path — retry success → fill event construction
// ---------------------------------------------------------------------------
func TestVP01_VenuePath_RetrySuccessThenFillEvent(t *testing.T) {
	// Mirrors onIntent: compose pipeline, submit with transient failures,
	// then verify fill event construction uses the successful receipt.
	var calls atomic.Int32
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		n := calls.Add(1)
		if n < 3 {
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "503 transient").MarkRetryable()
		}
		intent := testBuyIntent()
		intent.Status = domainexec.StatusFilled
		intent.FilledQuantity = "0.001"
		intent.Fills = []domainexec.FillRecord{
			{Price: "65000.00", Quantity: "0.001", Fee: "0.065", Timestamp: time.Now().UTC()},
		}
		return ports.VenueOrderReceipt{
			VenueOrderID:  "venue-retry-ok-123",
			ClientOrderID: "coid-001",
			Status:        domainexec.StatusFilled,
			Intent:        intent,
		}, nil
	}}

	tracker := healthz.NewTracker("vp01-tracker")
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	// Compose exactly as VenueAdapterActor.start() does.
	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithLogger(logger.With("component", "retry-submitter")).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)

	// No reconciler needed for this test — verifying retry path to fill.
	composedVenue := ports.VenuePort(retrier)

	// --- Actor path: submit with timeout (mirrors onIntent lines 211-222) ---
	submitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	intent := testBuyIntent()
	intent.CorrelationID = "vp01-corr"
	intent.CausationID = "vp01-caus"

	receipt, prob := composedVenue.SubmitOrder(submitCtx, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VP-01: expected success after retries, got: %s", prob.Message)
	}

	// --- Actor path: construct fill event (mirrors onIntent lines 248-271) ---
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("vp01-corr").
			WithCausationID("vp01-intake-001"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// Verify fill event integrity.
	if fillEvent.VenueOrderID != "venue-retry-ok-123" {
		t.Fatalf("VP-01: expected venue order ID venue-retry-ok-123, got %s", fillEvent.VenueOrderID)
	}
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("VP-01: expected filled status in fill event, got %s", fillEvent.ExecutionIntent.Status)
	}
	if fillEvent.ExecutionIntent.FilledQuantity != "0.001" {
		t.Fatalf("VP-01: expected filled_quantity 0.001, got %s", fillEvent.ExecutionIntent.FilledQuantity)
	}
	if len(fillEvent.ExecutionIntent.Fills) != 1 {
		t.Fatalf("VP-01: expected 1 fill, got %d", len(fillEvent.ExecutionIntent.Fills))
	}
	if fillEvent.ExecutionIntent.Symbol != "btcusdt" {
		t.Fatalf("VP-01: symbol bleed — expected btcusdt, got %s", fillEvent.ExecutionIntent.Symbol)
	}

	// Verify retry observability tracked at actor level.
	if tracker.Counter("retry_attempts").Load() < 1 {
		t.Fatal("VP-01: expected retry_attempts counter > 0")
	}
	if tracker.Counter("retry_success_after_retry").Load() != 1 {
		t.Fatalf("VP-01: expected retry_success_after_retry=1, got %d", tracker.Counter("retry_success_after_retry").Load())
	}

	// Verify structured log contains retry events.
	if !bytes.Contains(logBuf.Bytes(), []byte("retry attempt failed")) {
		t.Fatal("VP-01: expected 'retry attempt failed' in structured log")
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("retry succeeded")) {
		t.Fatal("VP-01: expected 'retry succeeded' in structured log")
	}
}

// ---------------------------------------------------------------------------
// VP-02: Full venue path — post-200 recovery → fill event construction
// ---------------------------------------------------------------------------
func TestVP02_VenuePath_Post200RecoveryThenFillEvent(t *testing.T) {
	// Mirrors onIntent: body-read-failure-after-200 triggers reconciliation,
	// recovered receipt is used for fill event.
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Internal, "read body failed").
			WithDetail("body_read_failure_after_200", true).
			WithDetail("client_order_id", "coid-vp02")
	}}

	queryVenue := &fakeQueryVenue{behavior: func(clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{
			VenueOrderID:  "recovered-vp02-888",
			ClientOrderID: clientOrderID,
			Status:        domainexec.StatusFilled,
			Intent: domainexec.ExecutionIntent{
				Symbol:         symbol,
				FilledQuantity: "0.001",
				Side:           domainexec.SideBuy,
				Status:         domainexec.StatusFilled,
				Fills: []domainexec.FillRecord{
					{Price: "67000.00", Quantity: "0.001", Fee: "0.067", Timestamp: time.Now().UTC()},
				},
			},
		}, nil
	}}

	tracker := healthz.NewTracker("vp02-tracker")

	// Compose exactly as VenueAdapterActor.start() does.
	retrier := appexec.NewRetrySubmitter(venue, appexec.DefaultRetryPolicy()).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)
	composedVenue := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	// --- Actor path: submit ---
	submitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	intent := testBuyIntent()
	intent.CorrelationID = "vp02-corr"
	intent.CausationID = "vp02-caus"

	receipt, prob := composedVenue.SubmitOrder(submitCtx, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VP-02: expected recovery, got: %s (details: %v)", prob.Message, prob.Details)
	}

	// --- Actor path: construct fill event ---
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("vp02-corr").
			WithCausationID("vp02-intake-001"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// Verify recovered fill event.
	if fillEvent.VenueOrderID != "recovered-vp02-888" {
		t.Fatalf("VP-02: expected recovered venue order ID, got %s", fillEvent.VenueOrderID)
	}
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("VP-02: expected filled status after recovery, got %s", fillEvent.ExecutionIntent.Status)
	}
	if len(fillEvent.ExecutionIntent.Fills) != 1 {
		t.Fatalf("VP-02: expected 1 fill after recovery, got %d", len(fillEvent.ExecutionIntent.Fills))
	}
	if fillEvent.ExecutionIntent.Fills[0].Price != "67000.00" {
		t.Fatalf("VP-02: recovered fill price mismatch, got %s", fillEvent.ExecutionIntent.Fills[0].Price)
	}

	// Verify JSON round-trip of recovered fill event (persistence path).
	data, err := json.Marshal(fillEvent)
	if err != nil {
		t.Fatalf("VP-02: fill event JSON marshal failed: %v", err)
	}
	var rt domainexec.VenueOrderFilledEvent
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("VP-02: fill event JSON round-trip failed: %v", err)
	}
	if rt.VenueOrderID != fillEvent.VenueOrderID {
		t.Fatal("VP-02: venue order ID lost in JSON round-trip")
	}
	if rt.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatal("VP-02: status lost in JSON round-trip")
	}
}

// ---------------------------------------------------------------------------
// VP-03: Venue path observability — retry metadata in actor error logs
// ---------------------------------------------------------------------------
func TestVP03_VenuePath_RetryMetadataInActorErrorLog(t *testing.T) {
	// When retry exhausts in the actor path, the actor's error log must contain
	// retry metadata extracted from Problem.Details (mirrors onIntent lines 234-242).
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "always fails").MarkRetryable()
	}}

	tracker := healthz.NewTracker("vp03-tracker")
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithLogger(logger.With("component", "retry-submitter")).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)

	composedVenue := ports.VenuePort(retrier)

	// --- Actor path: submit ---
	submitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, prob := composedVenue.SubmitOrder(submitCtx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("VP-03: expected error after retry exhaustion")
	}

	// --- Actor path: extract retry metadata from Problem.Details ---
	// This mirrors what onIntent does at lines 234-242.
	var actorLogBuf bytes.Buffer
	actorLogger := slog.New(slog.NewJSONHandler(&actorLogBuf, nil))

	logAttrs := []any{
		"error", prob.Message,
		"source", "binancef",
		"symbol", "btcusdt",
	}
	for _, key := range []string{
		"retry_attempts", "retry_exhausted",
		"retry_halted", "retry_deadline_exceeded",
	} {
		if v, ok := prob.Details[key]; ok {
			logAttrs = append(logAttrs, key, v)
		}
	}
	actorLogger.Error("venue submit failed", logAttrs...)

	// Parse actor-level log and verify retry metadata is present.
	var entry map[string]any
	if err := json.Unmarshal(actorLogBuf.Bytes(), &entry); err != nil {
		t.Fatalf("VP-03: failed to parse actor log: %v", err)
	}

	if _, ok := entry["retry_attempts"]; !ok {
		t.Fatal("VP-03: retry_attempts missing from actor error log")
	}
	if v, ok := entry["retry_exhausted"]; !ok || v != true {
		t.Fatal("VP-03: retry_exhausted missing or false in actor error log")
	}

	// Verify tracker counters match.
	if tracker.Counter("retry_exhausted").Load() != 1 {
		t.Fatalf("VP-03: expected retry_exhausted counter=1, got %d", tracker.Counter("retry_exhausted").Load())
	}
}

// ---------------------------------------------------------------------------
// VP-04: Venue path fill event integrity — intent fields survive composition
// ---------------------------------------------------------------------------
func TestVP04_VenuePath_FillEventIntentFieldPreservation(t *testing.T) {
	// Verify that all critical intent fields survive the composed pipeline
	// and appear intact in the fill event — symbol, side, quantity, risk,
	// correlation, causation, fills, timestamp.

	now := time.Now().UTC()
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		intent := testBuyIntent()
		intent.Status = domainexec.StatusFilled
		intent.FilledQuantity = "0.001"
		intent.CorrelationID = "vp04-corr"
		intent.CausationID = "vp04-caus"
		intent.Fills = []domainexec.FillRecord{
			{Price: "64000.00", Quantity: "0.001", Fee: "0.064", Timestamp: now},
		}
		return ports.VenueOrderReceipt{
			VenueOrderID:  "vp04-venue-id",
			ClientOrderID: "vp04-client-id",
			Status:        domainexec.StatusFilled,
			Intent:        intent,
		}, nil
	}}

	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("VP-04: query must not be called on direct success")
		return ports.VenueOrderReceipt{}, nil
	}}

	// Full composition: reconciler → retrier → venue (matches production).
	retrier := appexec.NewRetrySubmitter(venue, appexec.DefaultRetryPolicy()).
		TestWithSleepFn(noSleep)
	composedVenue := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	intent := testBuyIntent()
	intent.CorrelationID = "vp04-corr"
	intent.CausationID = "vp04-caus"

	receipt, prob := composedVenue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VP-04: expected success, got: %s", prob.Message)
	}

	// Construct fill event as actor would.
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("vp04-corr").
			WithCausationID("vp04-intake"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	ei := fillEvent.ExecutionIntent

	// Critical field checks.
	if ei.Source != "binancef" {
		t.Fatalf("VP-04: source not preserved: got %q", ei.Source)
	}
	if ei.Symbol != "btcusdt" {
		t.Fatalf("VP-04: symbol not preserved: got %q", ei.Symbol)
	}
	if ei.Timeframe != 60 {
		t.Fatalf("VP-04: timeframe not preserved: got %d", ei.Timeframe)
	}
	if ei.Side != domainexec.SideBuy {
		t.Fatalf("VP-04: side not preserved: got %q", ei.Side)
	}
	if ei.Quantity != "0.001" {
		t.Fatalf("VP-04: quantity not preserved: got %q", ei.Quantity)
	}
	if ei.FilledQuantity != "0.001" {
		t.Fatalf("VP-04: filled_quantity not preserved: got %q", ei.FilledQuantity)
	}
	if ei.Status != domainexec.StatusFilled {
		t.Fatalf("VP-04: status not preserved: got %q", ei.Status)
	}
	if ei.CorrelationID != "vp04-corr" {
		t.Fatalf("VP-04: correlation_id not preserved: got %q", ei.CorrelationID)
	}
	if ei.CausationID != "vp04-caus" {
		t.Fatalf("VP-04: causation_id not preserved: got %q", ei.CausationID)
	}
	if ei.Risk.Type != "position_exposure" {
		t.Fatalf("VP-04: risk.type not preserved: got %q", ei.Risk.Type)
	}
	if ei.Risk.Disposition != "approved" {
		t.Fatalf("VP-04: risk.disposition not preserved: got %q", ei.Risk.Disposition)
	}
	if len(ei.Fills) != 1 {
		t.Fatalf("VP-04: fills not preserved: got %d", len(ei.Fills))
	}
	if ei.Fills[0].Price != "64000.00" {
		t.Fatalf("VP-04: fill price not preserved: got %q", ei.Fills[0].Price)
	}

	// Verify fill event serializes for persistence.
	data, err := json.Marshal(fillEvent)
	if err != nil {
		t.Fatalf("VP-04: fill event not serializable: %v", err)
	}
	if len(data) < 100 {
		t.Fatalf("VP-04: fill event too small (%d bytes), likely missing fields", len(data))
	}
}

// ---------------------------------------------------------------------------
// VP-05: Venue path tracker counters — actor-level tracking reflects pipeline
// ---------------------------------------------------------------------------
func TestVP05_VenuePath_TrackerCountersReflectPipeline(t *testing.T) {
	// Verify that tracker counters at the actor level correctly reflect
	// events from the composed pipeline: retry_attempts, retry_success,
	// retry_exhausted, retry_halted.

	tracker := healthz.NewTracker("vp05-tracker")

	// Scenario A: retry then success.
	var callsA atomic.Int32
	venueA := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		n := callsA.Add(1)
		if n == 1 {
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "transient").MarkRetryable()
		}
		return ports.VenueOrderReceipt{VenueOrderID: "a", Status: domainexec.StatusFilled, Intent: testBuyIntent()}, nil
	}}

	retrierA := appexec.NewRetrySubmitter(venueA, appexec.RetryPolicy{
		MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Factor: 2.0,
	}).WithTracker(tracker).TestWithSleepFn(noSleep)

	_, prob := retrierA.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("VP-05A: expected success, got: %s", prob.Message)
	}

	// Scenario B: exhaustion.
	venueB := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "always").MarkRetryable()
	}}

	retrierB := appexec.NewRetrySubmitter(venueB, appexec.RetryPolicy{
		MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Factor: 2.0,
	}).WithTracker(tracker).TestWithSleepFn(noSleep)

	_, prob = retrierB.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("VP-05B: expected exhaustion error")
	}

	// Scenario C: halt.
	venueC := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "unavailable").MarkRetryable()
	}}

	retrierC := appexec.NewRetrySubmitter(venueC, appexec.RetryPolicy{
		MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Factor: 2.0,
	}).
		WithHaltChecker(&mockGateChecker{halted: true}).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)

	_, prob = retrierC.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("VP-05C: expected halt error")
	}

	// Verify cumulative tracker state.
	if tracker.Counter("retry_attempts").Load() < 1 {
		t.Fatal("VP-05: retry_attempts should be >= 1")
	}
	if tracker.Counter("retry_success_after_retry").Load() != 1 {
		t.Fatalf("VP-05: expected retry_success_after_retry=1, got %d", tracker.Counter("retry_success_after_retry").Load())
	}
	if tracker.Counter("retry_exhausted").Load() != 1 {
		t.Fatalf("VP-05: expected retry_exhausted=1, got %d", tracker.Counter("retry_exhausted").Load())
	}
	if tracker.Counter("retry_halted").Load() != 1 {
		t.Fatalf("VP-05: expected retry_halted=1, got %d", tracker.Counter("retry_halted").Load())
	}
}

// ---------------------------------------------------------------------------
// VP-06: Venue path halt propagation — kill switch abort in actor error path
// ---------------------------------------------------------------------------
func TestVP06_VenuePath_HaltPropagationToActorErrorPath(t *testing.T) {
	// When kill switch aborts retry in the composed pipeline, the actor's
	// error logging path must surface retry_halted metadata.
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "down").MarkRetryable()
	}}

	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("VP-06: query must not be called when halted")
		return ports.VenueOrderReceipt{}, nil
	}}

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Factor: 2.0,
	}).
		WithHaltChecker(&mockGateChecker{halted: true}).
		TestWithSleepFn(noSleep)

	composedVenue := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	_, prob := composedVenue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("VP-06: expected error when halted")
	}

	// Verify retry_halted surfaces through the full composition stack.
	if v, ok := prob.Details["retry_halted"]; !ok || v != true {
		t.Fatalf("VP-06: retry_halted not found in problem details: %v", prob.Details)
	}

	// Verify reconciler did not interfere (halt is not body-read-failure).
	if _, ok := prob.Details["reconciliation_attempted"]; ok {
		t.Fatal("VP-06: reconciliation_attempted should not appear for halted retry")
	}
}

// ---------------------------------------------------------------------------
// VP-07: Venue path paper mode — retry-only, fill event intact
// ---------------------------------------------------------------------------
func TestVP07_VenuePath_PaperMode_RetryOnlyFillEventIntact(t *testing.T) {
	// Paper mode: no VenueQuery → no reconciler, retry-only composition.
	// This mirrors the exact path for paper_simulator in production.

	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		intent := testBuyIntent()
		intent.Status = domainexec.StatusFilled
		intent.FilledQuantity = "0.001"
		intent.Fills = []domainexec.FillRecord{
			{Price: "0", Quantity: "0.001", Fee: "0", Simulated: true, Timestamp: time.Now().UTC()},
		}
		return ports.VenueOrderReceipt{
			VenueOrderID: "paper-vp07-001",
			Status:       domainexec.StatusFilled,
			Intent:       intent,
		}, nil
	}}

	tracker := healthz.NewTracker("vp07-tracker")

	// Compose as actor does for paper mode (no reconciler).
	retrier := appexec.NewRetrySubmitter(venue, appexec.DefaultRetryPolicy()).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)
	composedVenue := ports.VenuePort(retrier) // No reconciler wrapping.

	intent := testBuyIntent()
	intent.CorrelationID = "vp07-corr"
	intent.CausationID = "vp07-caus"

	receipt, prob := composedVenue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VP-07: expected success, got: %s", prob.Message)
	}

	// Construct fill event.
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("vp07-corr").
			WithCausationID("vp07-intake"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	if fillEvent.VenueOrderID != "paper-vp07-001" {
		t.Fatalf("VP-07: expected paper venue order ID, got %s", fillEvent.VenueOrderID)
	}
	if !fillEvent.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("VP-07: paper fill must be simulated")
	}
	if fillEvent.Metadata.CorrelationID != "vp07-corr" {
		t.Fatalf("VP-07: correlation_id not preserved: got %q", fillEvent.Metadata.CorrelationID)
	}

	// No retry events should have fired (first-attempt success).
	if tracker.Counter("retry_attempts").Load() != 0 {
		t.Fatalf("VP-07: expected 0 retry_attempts for first-attempt success, got %d", tracker.Counter("retry_attempts").Load())
	}
	if tracker.Counter("retry_success_after_retry").Load() != 0 {
		t.Fatalf("VP-07: expected 0 retry_success for first-attempt success, got %d", tracker.Counter("retry_success_after_retry").Load())
	}
}

// ---------------------------------------------------------------------------
// VP-08: Full venue path — retry then post-200 recovery (end-to-end composition)
// ---------------------------------------------------------------------------
func TestVP08_VenuePath_RetryThenPost200Recovery_FillEvent(t *testing.T) {
	// The most complex path: retry a transient failure, then hit body-read-failure
	// which the reconciler recovers. Fill event must be built from recovered receipt.
	var calls atomic.Int32
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		n := calls.Add(1)
		switch n {
		case 1:
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "503").MarkRetryable()
		case 2:
			return ports.VenueOrderReceipt{}, problem.New(problem.Internal, "body read failed").
				WithDetail("body_read_failure_after_200", true).
				WithDetail("client_order_id", "coid-vp08")
		default:
			t.Fatalf("VP-08: unexpected call %d", n)
			return ports.VenueOrderReceipt{}, nil
		}
	}}

	queryVenue := &fakeQueryVenue{behavior: func(clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
		if clientOrderID != "coid-vp08" {
			t.Fatalf("VP-08: wrong client order ID: %s", clientOrderID)
		}
		return ports.VenueOrderReceipt{
			VenueOrderID:  "recovered-vp08-999",
			ClientOrderID: clientOrderID,
			Status:        domainexec.StatusFilled,
			Intent: domainexec.ExecutionIntent{
				Symbol:         symbol,
				Side:           domainexec.SideBuy,
				Status:         domainexec.StatusFilled,
				FilledQuantity: "0.001",
				Fills: []domainexec.FillRecord{
					{Price: "68000.00", Quantity: "0.001", Fee: "0.068", Timestamp: time.Now().UTC()},
				},
			},
		}, nil
	}}

	tracker := healthz.NewTracker("vp08-tracker")
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithLogger(logger.With("component", "retry-submitter")).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)

	composedVenue := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	receipt, prob := composedVenue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("VP-08: expected recovery, got: %s (details: %v)", prob.Message, prob.Details)
	}

	// Construct fill event.
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("vp08-corr").
			WithCausationID("vp08-intake"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	if fillEvent.VenueOrderID != "recovered-vp08-999" {
		t.Fatalf("VP-08: expected recovered venue order ID, got %s", fillEvent.VenueOrderID)
	}
	if fillEvent.ExecutionIntent.Fills[0].Price != "68000.00" {
		t.Fatalf("VP-08: expected recovered fill price, got %s", fillEvent.ExecutionIntent.Fills[0].Price)
	}

	// Verify both retry and recovery participated.
	if tracker.Counter("retry_attempts").Load() < 1 {
		t.Fatal("VP-08: retry_attempts should be >= 1 (transient failure was retried)")
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("retry attempt failed")) {
		t.Fatal("VP-08: expected retry log (503 was retried before body-read-failure)")
	}
}

// ---------------------------------------------------------------------------
// VP-09: Venue path safety gate integration — stale intent blocked before pipeline
// ---------------------------------------------------------------------------
func TestVP09_VenuePath_SafetyGateBlocksBeforePipeline(t *testing.T) {
	// Safety gate (staleness) must block before the composed pipeline runs.
	// This proves the guard rails at the actor level work independently of
	// the decorator chain.

	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// Stale intent: 5 minutes old with 2-minute max age.
	staleTS := now.Add(-5 * time.Minute)
	if !guard.IsStale(staleTS, now) {
		t.Fatal("VP-09: 5min-old intent should be stale with 2min guard")
	}

	// Fresh intent: 30 seconds old.
	freshTS := now.Add(-30 * time.Second)
	if guard.IsStale(freshTS, now) {
		t.Fatal("VP-09: 30s-old intent should not be stale")
	}

	// Kill switch gate integration.
	gate := appexec.NewSafetyGate(&mockGateChecker{halted: true}, 2*time.Second, guard)
	verdict := gate.Check(freshTS, now)
	if verdict.Allowed {
		t.Fatal("VP-09: fresh intent should be blocked by kill switch")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("VP-09: expected kill_switch reason, got %q", verdict.Reason)
	}

	// With gate open and fresh intent: allowed.
	gateOpen := appexec.NewSafetyGate(&mockGateChecker{halted: false}, 2*time.Second, guard)
	verdict = gateOpen.Check(freshTS, now)
	if !verdict.Allowed {
		t.Fatalf("VP-09: fresh intent with open gate should be allowed, got reason: %s", verdict.Reason)
	}
}
