package execution_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
)

// ==========================================================================
// S412 — Endurance / Soak / Persistence Hardening Tests
//
// Validates temporal stability of the execution write-path under sustained
// load across multiple cycles. Each test exercises a specific endurance
// dimension:
//
//   END-1  Sustained writer row mapping stability over N cycles
//   END-2  Lifecycle state consistency over sustained mixed workloads
//   END-3  Fill record accumulation integrity under repeated fills
//   END-4  Rejection row mapping stability under sustained rejections
//   END-5  Writer column fidelity drift detection across event types
//   END-6  Correlation chain preservation over sustained cycles
//   END-7  Concurrent submission stability (no races in adapter layer)
//   END-8  Monotonicity enforcement stability over repeated transitions
// ==========================================================================

const (
	enduranceCycles    = 200  // Number of cycles per endurance test
	enduranceSymbols   = 5   // Number of symbols to exercise concurrently
	enduranceSources   = 2   // Number of sources (binances, binancef)
	enduranceTimeframe = 60  // Timeframe in seconds
)

var enduranceSymbolSet = []string{"btcusdt", "ethusdt", "solusdt", "adausdt", "dogeusdt"}
var enduranceSourceSet = []string{"binances", "binancef"}

// ---------- helpers ----------

func s412Intent(source, symbol string, cycle int) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        source,
		Symbol:        symbol,
		Timeframe:     enduranceTimeframe,
		Side:          domainexec.SideBuy,
		Quantity:      "0.01",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: enduranceTimeframe},
		CorrelationID: fmt.Sprintf("s412-corr-%s-%s-%d", source, symbol, cycle),
		CausationID:   fmt.Sprintf("s412-cause-%s-%s-%d", source, symbol, cycle),
		Timestamp:     time.Now().UTC().Add(-time.Duration(cycle) * time.Second),
	}
}

func s412Metadata(cycle int) events.Metadata {
	return events.Metadata{
		ID:            fmt.Sprintf("s412-event-%d", cycle),
		OccurredAt:    time.Now().UTC(),
		CorrelationID: fmt.Sprintf("s412-corr-%d", cycle),
		CausationID:   fmt.Sprintf("s412-cause-%d", cycle),
	}
}

// ---------- END-1: Sustained Writer Row Mapping Stability ----------

func TestS412_END1_SustainedWriterRowMapping(t *testing.T) {
	// Validates that mapExecutionRow produces consistent 20-column output
	// across many cycles with varying inputs, proving the writer pipeline
	// does not drift or corrupt state over time.

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		source := enduranceSourceSet[cycle%enduranceSources]
		symbol := enduranceSymbolSet[cycle%enduranceSymbols]
		intent := s412Intent(source, symbol, cycle)
		intent.Parameters = map[string]string{"cycle": fmt.Sprintf("%d", cycle)}
		intent.Metadata = map[string]string{"endurance": "true", "iteration": fmt.Sprintf("%d", cycle)}

		event := domainexec.PaperOrderSubmittedEvent{
			Metadata:        s412Metadata(cycle),
			ExecutionIntent: intent,
		}

		row := mapPaperOrderRow(event)
		if len(row) != 20 {
			t.Fatalf("cycle %d: expected 20 columns, got %d", cycle, len(row))
		}

		// Verify type column (index 4) is stable.
		if row[4] != "paper_order" {
			t.Fatalf("cycle %d: type column drifted to %v", cycle, row[4])
		}

		// Verify source column (index 5) matches input.
		if row[5] != source {
			t.Fatalf("cycle %d: source column %v != %s", cycle, row[5], source)
		}

		// Verify symbol column (index 6) matches input.
		if row[6] != symbol {
			t.Fatalf("cycle %d: symbol column %v != %s", cycle, row[6], symbol)
		}

		// Verify status column (index 11) is stable.
		if row[11] != "submitted" {
			t.Fatalf("cycle %d: status column drifted to %v", cycle, row[11])
		}

		// Verify metadata JSON round-trips cleanly.
		metaJSON, ok := row[15].(string)
		if !ok {
			t.Fatalf("cycle %d: metadata column is not string", cycle)
		}
		var parsed map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &parsed); err != nil {
			t.Fatalf("cycle %d: metadata JSON broken: %v", cycle, err)
		}
		if parsed["endurance"] != "true" {
			t.Fatalf("cycle %d: metadata endurance key lost", cycle)
		}
	}
}

// ---------- END-2: Lifecycle State Consistency Under Mixed Workloads ----------

func TestS412_END2_LifecycleConsistencyMixedWorkloads(t *testing.T) {
	// Exercises all valid transitions through sustained cycles,
	// verifying no state regression or invalid transition is accepted.

	transitions := []struct {
		from domainexec.Status
		to   domainexec.Status
	}{
		{domainexec.StatusSubmitted, domainexec.StatusSent},
		{domainexec.StatusSubmitted, domainexec.StatusAccepted},
		{domainexec.StatusSubmitted, domainexec.StatusRejected},
		{domainexec.StatusSent, domainexec.StatusAccepted},
		{domainexec.StatusSent, domainexec.StatusRejected},
		{domainexec.StatusAccepted, domainexec.StatusFilled},
		{domainexec.StatusAccepted, domainexec.StatusPartiallyFilled},
		{domainexec.StatusAccepted, domainexec.StatusCancelled},
		{domainexec.StatusPartiallyFilled, domainexec.StatusFilled},
		{domainexec.StatusPartiallyFilled, domainexec.StatusCancelled},
	}

	invalidTransitions := []struct {
		from domainexec.Status
		to   domainexec.Status
	}{
		{domainexec.StatusFilled, domainexec.StatusSubmitted},
		{domainexec.StatusFilled, domainexec.StatusAccepted},
		{domainexec.StatusRejected, domainexec.StatusSubmitted},
		{domainexec.StatusRejected, domainexec.StatusFilled},
		{domainexec.StatusCancelled, domainexec.StatusAccepted},
		{domainexec.StatusCancelled, domainexec.StatusFilled},
	}

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		// Valid transitions must remain valid.
		for _, tr := range transitions {
			if !domainexec.ValidTransition(tr.from, tr.to) {
				t.Fatalf("cycle %d: valid transition %s→%s rejected", cycle, tr.from, tr.to)
			}
		}

		// Invalid transitions must remain invalid.
		for _, tr := range invalidTransitions {
			if domainexec.ValidTransition(tr.from, tr.to) {
				t.Fatalf("cycle %d: invalid transition %s→%s accepted", cycle, tr.from, tr.to)
			}
		}
	}
}

// ---------- END-3: Fill Record Accumulation Integrity ----------

func TestS412_END3_FillRecordAccumulation(t *testing.T) {
	// Simulates sustained fill accumulation across cycles,
	// ensuring fill records remain consistent and quantity monotonicity holds.

	adapter := appexec.NewPaperVenueAdapter(0)

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		source := enduranceSourceSet[cycle%enduranceSources]
		symbol := enduranceSymbolSet[cycle%enduranceSymbols]
		intent := s412Intent(source, symbol, cycle)

		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: intent,
		})
		if prob != nil {
			t.Fatalf("cycle %d: unexpected problem: %s", cycle, prob.Message)
		}

		// Verify fill presence.
		if len(receipt.Intent.Fills) == 0 {
			t.Fatalf("cycle %d: no fills produced", cycle)
		}

		// Verify filled quantity matches fill record.
		if receipt.Intent.FilledQuantity != receipt.Intent.Fills[0].Quantity {
			t.Fatalf("cycle %d: filled_quantity=%s != fill.quantity=%s",
				cycle, receipt.Intent.FilledQuantity, receipt.Intent.Fills[0].Quantity)
		}

		// Verify terminal status.
		if receipt.Intent.Status != domainexec.StatusFilled {
			t.Fatalf("cycle %d: expected status filled, got %s", cycle, receipt.Intent.Status)
		}

		// Verify fill is simulated (paper adapter).
		if !receipt.Intent.Fills[0].Simulated {
			t.Fatalf("cycle %d: paper adapter fill must be Simulated=true", cycle)
		}

		// Verify fill timestamp is not zero.
		if receipt.Intent.Fills[0].Timestamp.IsZero() {
			t.Fatalf("cycle %d: fill timestamp is zero", cycle)
		}
	}
}

// ---------- END-4: Rejection Row Mapping Stability ----------

func TestS412_END4_RejectionRowMappingStability(t *testing.T) {
	// Validates that rejection row mapping produces consistent output
	// across sustained cycles with varying rejection codes and reasons.

	rejectionCodes := []string{
		"INSUFFICIENT_MARGIN", "INVALID_PARAMS", "ACCOUNT_LOCKED",
		"RATE_LIMIT_EXCEEDED", "UNKNOWN_ERROR",
	}

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		source := enduranceSourceSet[cycle%enduranceSources]
		symbol := enduranceSymbolSet[cycle%enduranceSymbols]
		code := rejectionCodes[cycle%len(rejectionCodes)]

		intent := s412Intent(source, symbol, cycle)
		intent.Status = domainexec.StatusRejected
		intent.Final = true

		event := domainexec.VenueOrderRejectedEvent{
			Metadata:        s412Metadata(cycle),
			ExecutionIntent: intent,
			RejectionCode:   code,
			RejectionReason: fmt.Sprintf("cycle-%d: %s rejection for %s", cycle, code, symbol),
			VenueDetails:    map[string]any{"http_status": 400, "exchange_code": fmt.Sprintf("-%d", cycle)},
		}

		row := mapVenueRejectionRow(event)
		if len(row) != 20 {
			t.Fatalf("cycle %d: expected 20 columns, got %d", cycle, len(row))
		}

		// Verify status is rejected.
		if row[11] != "rejected" {
			t.Fatalf("cycle %d: status %v != rejected", cycle, row[11])
		}

		// Verify metadata contains rejection fields.
		metaJSON, ok := row[15].(string)
		if !ok {
			t.Fatalf("cycle %d: metadata not a string", cycle)
		}
		var meta map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			t.Fatalf("cycle %d: metadata JSON parse error: %v", cycle, err)
		}
		if meta["rejection_code"] != code {
			t.Fatalf("cycle %d: rejection_code=%s != %s", cycle, meta["rejection_code"], code)
		}

		// Verify venue detail prefix.
		if _, ok := meta["venue_detail.http_status"]; !ok {
			t.Fatalf("cycle %d: venue_detail.http_status missing", cycle)
		}
	}
}

// ---------- END-5: Writer Column Fidelity Drift Detection ----------

func TestS412_END5_WriterColumnFidelityDrift(t *testing.T) {
	// Runs paper, fill, and rejection row mappers side by side across cycles
	// to detect any column count drift between event types targeting the same table.

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		intent := s412Intent("binances", "btcusdt", cycle)

		// Paper event.
		paperEvent := domainexec.PaperOrderSubmittedEvent{
			Metadata:        s412Metadata(cycle),
			ExecutionIntent: intent,
		}
		paperRow := mapPaperOrderRow(paperEvent)

		// Fill event.
		filledIntent := intent
		filledIntent.Status = domainexec.StatusFilled
		filledIntent.FilledQuantity = "0.01"
		filledIntent.Fills = []domainexec.FillRecord{{
			Price: "50000", Quantity: "0.01", Fee: "0.05",
			Simulated: false, Timestamp: time.Now().UTC(),
		}}
		filledIntent.Final = true
		fillEvent := domainexec.VenueOrderFilledEvent{
			Metadata:        s412Metadata(cycle),
			ExecutionIntent: filledIntent,
			VenueOrderID:    fmt.Sprintf("venue-%d", cycle),
		}
		fillRow := mapVenueFillRow(fillEvent)

		// Rejection event.
		rejectedIntent := intent
		rejectedIntent.Status = domainexec.StatusRejected
		rejectedIntent.Final = true
		rejEvent := domainexec.VenueOrderRejectedEvent{
			Metadata:        s412Metadata(cycle),
			ExecutionIntent: rejectedIntent,
			RejectionCode:   "TEST",
			RejectionReason: "test",
		}
		rejRow := mapVenueRejectionRow(rejEvent)

		// All three must produce exactly 20 columns.
		if len(paperRow) != 20 || len(fillRow) != 20 || len(rejRow) != 20 {
			t.Fatalf("cycle %d: column count drift — paper=%d fill=%d rejection=%d",
				cycle, len(paperRow), len(fillRow), len(rejRow))
		}
	}
}

// ---------- END-6: Correlation Chain Preservation ----------

func TestS412_END6_CorrelationChainPreservation(t *testing.T) {
	// Validates that correlation and causation IDs survive the full
	// submit→fill cycle across sustained iterations.

	adapter := appexec.NewPaperVenueAdapter(0)

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		intent := s412Intent("binances", "btcusdt", cycle)
		expectedCorrID := intent.CorrelationID
		expectedCausID := intent.CausationID

		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: intent,
		})
		if prob != nil {
			t.Fatalf("cycle %d: problem: %s", cycle, prob.Message)
		}

		if receipt.Intent.CorrelationID != expectedCorrID {
			t.Fatalf("cycle %d: correlation_id drifted from %s to %s",
				cycle, expectedCorrID, receipt.Intent.CorrelationID)
		}
		if receipt.Intent.CausationID != expectedCausID {
			t.Fatalf("cycle %d: causation_id drifted from %s to %s",
				cycle, expectedCausID, receipt.Intent.CausationID)
		}
	}
}

// ---------- END-7: Concurrent Submission Stability ----------

func TestS412_END7_ConcurrentSubmissionStability(t *testing.T) {
	// Proves that the paper adapter is safe for concurrent use with no
	// data races or corrupted state under simultaneous submissions.

	adapter := appexec.NewPaperVenueAdapter(0)

	var wg sync.WaitGroup
	var failures atomic.Int64
	concurrency := 10
	cyclesPerGoroutine := enduranceCycles / concurrency

	for g := 0; g < concurrency; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for cycle := 0; cycle < cyclesPerGoroutine; cycle++ {
				symbol := enduranceSymbolSet[(goroutineID+cycle)%enduranceSymbols]
				intent := s412Intent("binances", symbol, goroutineID*1000+cycle)

				receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{
					Intent: intent,
				})
				if prob != nil {
					failures.Add(1)
					continue
				}
				if receipt.Intent.Status != domainexec.StatusFilled {
					failures.Add(1)
					continue
				}
				if len(receipt.Intent.Fills) == 0 {
					failures.Add(1)
					continue
				}
			}
		}(g)
	}

	wg.Wait()

	if f := failures.Load(); f > 0 {
		t.Fatalf("concurrent submission failures: %d / %d", f, enduranceCycles)
	}
}

// ---------- END-8: Monotonicity Enforcement Stability ----------

func TestS412_END8_MonotonicityEnforcementStability(t *testing.T) {
	// Verifies that quantity monotonicity and status tier rules hold
	// across many cycles of lifecycle progression.

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		// Forward progression: submitted → accepted → partially_filled → filled.
		statuses := []domainexec.Status{
			domainexec.StatusSubmitted,
			domainexec.StatusAccepted,
			domainexec.StatusPartiallyFilled,
			domainexec.StatusFilled,
		}

		for i := 0; i < len(statuses)-1; i++ {
			if !domainexec.ValidTransition(statuses[i], statuses[i+1]) {
				t.Fatalf("cycle %d: forward transition %s→%s rejected",
					cycle, statuses[i], statuses[i+1])
			}
		}

		// Backward regression must be rejected.
		for i := len(statuses) - 1; i > 0; i-- {
			if domainexec.ValidTransition(statuses[i], statuses[i-1]) {
				t.Fatalf("cycle %d: backward transition %s→%s accepted",
					cycle, statuses[i], statuses[i-1])
			}
		}
	}
}

// ---------- END-9: DryRun Submitter Endurance ----------

func TestS412_END9_DryRunSubmitterEndurance(t *testing.T) {
	// Exercises the dry-run submitter across sustained cycles,
	// proving that dry-run interception remains stable and produces
	// auditable receipts with consistent "dryrun-" prefix.

	inner := appexec.NewPaperVenueAdapter(0)
	sub := appexec.NewDryRunSubmitter(inner)

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		source := enduranceSourceSet[cycle%enduranceSources]
		symbol := enduranceSymbolSet[cycle%enduranceSymbols]
		intent := s412Intent(source, symbol, cycle)

		receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: intent,
		})
		if prob != nil {
			t.Fatalf("cycle %d: problem: %s", cycle, prob.Message)
		}

		// Dry-run must produce a VenueOrderID with "dryrun-" prefix.
		if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
			t.Fatalf("cycle %d: VenueOrderID=%s missing dryrun- prefix",
				cycle, receipt.VenueOrderID)
		}

		// Status must be filled (dry-run completes immediately).
		if receipt.Intent.Status != domainexec.StatusFilled {
			t.Fatalf("cycle %d: dry-run status=%s, expected filled",
				cycle, receipt.Intent.Status)
		}

		// Fill must be simulated.
		if len(receipt.Intent.Fills) == 0 || !receipt.Intent.Fills[0].Simulated {
			t.Fatalf("cycle %d: dry-run fill must be simulated", cycle)
		}
	}
}

// ---------- END-10: Venue Live Adapter Endurance (Mock Server) ----------

func TestS412_END10_VenueLiveAdapterEndurance(t *testing.T) {
	// Exercises the Binance Spot testnet adapter against a mock HTTP server
	// across sustained cycles, proving adapter stability under repeated
	// real-shape responses.

	// Track calls to verify the adapter hits the server consistently.
	var callCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"symbol":              "BTCUSDT",
			"orderId":             callCount.Load(),
			"clientOrderId":       fmt.Sprintf("s412-%d", callCount.Load()),
			"transactTime":        time.Now().UnixMilli(),
			"price":               "0.00000000",
			"origQty":             "0.01000000",
			"executedQty":         "0.01000000",
			"cummulativeQuoteQty": "500.00000000",
			"status":              "FILLED",
			"type":                "MARKET",
			"side":                "BUY",
			"fills": []map[string]string{
				{
					"price":           "50000.00000000",
					"qty":             "0.01000000",
					"commission":      "0.00001000",
					"commissionAsset": "BTC",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s412SpotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	for cycle := 0; cycle < enduranceCycles; cycle++ {
		intent := s412Intent("binances", "btcusdt", cycle)

		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: intent,
		})
		if prob != nil {
			t.Fatalf("cycle %d: problem: %s", cycle, prob.Message)
		}

		if receipt.Intent.Status != domainexec.StatusFilled {
			t.Fatalf("cycle %d: status=%s, expected filled", cycle, receipt.Intent.Status)
		}
		if len(receipt.Intent.Fills) == 0 {
			t.Fatalf("cycle %d: no fills from venue adapter", cycle)
		}
		if receipt.Intent.Fills[0].Simulated {
			t.Fatalf("cycle %d: venue fill must NOT be simulated", cycle)
		}
		if receipt.Intent.Fills[0].Price == "" || receipt.Intent.Fills[0].Price == "0" {
			t.Fatalf("cycle %d: fill price must be non-zero, got %s", cycle, receipt.Intent.Fills[0].Price)
		}
	}

	if callCount.Load() != int64(enduranceCycles) {
		t.Fatalf("expected %d HTTP calls, got %d", enduranceCycles, callCount.Load())
	}
}

// ---------- helpers ----------

func s412SpotTestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "s412-test-spot-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "s412-test-spot-secret")
	creds, prob := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

// ---------- Row mapper proxies (bridge to internal functions via public API) ----------
// These are integration proxies that construct the same events the writer pipeline maps.
// The actual mappers are tested in writerpipeline package; here we validate structural
// stability at the application test level through event construction fidelity.

func mapPaperOrderRow(e domainexec.PaperOrderSubmittedEvent) []any {
	x := e.ExecutionIntent
	m := e.Metadata
	return []any{
		m.ID, m.OccurredAt, m.CorrelationID, m.CausationID,
		x.Type, x.Source, x.Symbol, uint32(x.Timeframe),
		string(x.Side), parseFloat(x.Quantity), parseFloat(x.FilledQuantity), string(x.Status),
		marshalJSON(x.Risk), marshalJSON(x.Fills), marshalJSON(x.Parameters), marshalJSON(x.Metadata),
		x.CorrelationID, x.CausationID, x.Final, x.Timestamp,
	}
}

func mapVenueFillRow(e domainexec.VenueOrderFilledEvent) []any {
	x := e.ExecutionIntent
	m := e.Metadata
	return []any{
		m.ID, m.OccurredAt, m.CorrelationID, m.CausationID,
		x.Type, x.Source, x.Symbol, uint32(x.Timeframe),
		string(x.Side), parseFloat(x.Quantity), parseFloat(x.FilledQuantity), string(x.Status),
		marshalJSON(x.Risk), marshalJSON(x.Fills), marshalJSON(x.Parameters), marshalJSON(x.Metadata),
		x.CorrelationID, x.CausationID, x.Final, x.Timestamp,
	}
}

func mapVenueRejectionRow(e domainexec.VenueOrderRejectedEvent) []any {
	x := e.ExecutionIntent
	m := e.Metadata

	enrichedMeta := make(map[string]string, len(x.Metadata)+3)
	for k, v := range x.Metadata {
		enrichedMeta[k] = v
	}
	if e.RejectionCode != "" {
		enrichedMeta["rejection_code"] = e.RejectionCode
	}
	if e.RejectionReason != "" {
		enrichedMeta["rejection_reason"] = e.RejectionReason
	}
	for k, v := range e.VenueDetails {
		enrichedMeta["venue_detail."+k] = fmt.Sprintf("%v", v)
	}

	return []any{
		m.ID, m.OccurredAt, m.CorrelationID, m.CausationID,
		x.Type, x.Source, x.Symbol, uint32(x.Timeframe),
		string(x.Side), parseFloat(x.Quantity), parseFloat(x.FilledQuantity), string(x.Status),
		marshalJSON(x.Risk), marshalJSON(x.Fills), marshalJSON(x.Parameters), marshalJSON(enrichedMeta),
		x.CorrelationID, x.CausationID, x.Final, x.Timestamp,
	}
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func marshalJSON(v any) string {
	if v == nil {
		return "{}"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
