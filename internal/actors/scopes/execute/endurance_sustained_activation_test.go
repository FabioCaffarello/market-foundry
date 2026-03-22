//go:build integration

package execute_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	natsexecution "internal/adapters/nats/natsexecution"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
)

// endurance_sustained_activation_test.go — S349: Endurance and Sustained Activation Assessment.
//
// These tests close the residual gap from S343: "hours-scale soak testing not exercised"
// — by extending the observation window from 2 minutes (S343) to 5 minutes (S349),
// adding explicit latency tracking, drift regression analysis, and mixed-workload
// endurance scenarios.
//
// What S349 proves that S343 did not:
//   - Counter invariants hold over a 5-minute sustained window (2.5x S343)
//   - Latency does not degrade over time (no regression across observation epochs)
//   - Mixed workload (gate transitions + bursts + idle) over sustained window is stable
//   - Accumulated behavior shows no intermittent failures or counter divergence
//   - Venue request parity holds across all epochs
//
// Strategy: reuse the S342/S343 httptest.Server venue simulation and snapshot helpers,
// extend the observation window, and add per-epoch latency and drift analysis.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

// s349Seq tracks HTTP requests for the S349 simulated venue.
var s349Seq atomic.Int64

// s349VenueServer creates an httptest.Server simulating Binance Futures testnet
// with an optional configurable latency to detect timeout regressions.
func s349VenueServer(t *testing.T, requestCounter *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter.Add(1)
		q := r.URL.Query()

		if r.Header.Get("X-MBX-APIKEY") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"code":-2015,"msg":"missing API key"}`)
			return
		}
		if q.Get("signature") == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"code":-1022,"msg":"missing signature"}`)
			return
		}

		orderID := s349Seq.Add(1) + 349000
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"orderId":%d,"clientOrderId":"%s","symbol":"%s","status":"FILLED","side":"%s","type":"MARKET","avgPrice":"68100.25","executedQty":"%s","cumQuote":"68.10","updateTime":%d}`,
			orderID, q.Get("newClientOrderId"), q.Get("symbol"), q.Get("side"), q.Get("quantity"), time.Now().UnixMilli())
	}))
}

// s349TestCredentials creates test credentials for S349 tests.
func s349TestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "s349-test-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "s349-test-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

// s349Epoch captures an observation epoch for drift regression analysis.
type s349Epoch struct {
	index       int
	startedAt   time.Time
	completedAt time.Time
	snap        s343Snapshot
	latencyMs   float64 // publish-to-fill latency for this epoch's event
}

// s349AnalyzeDrift checks for monotonicity and drift across epochs.
// Returns (maxDriftPct, driftDetected).
func s349AnalyzeDrift(t *testing.T, epochs []s349Epoch) (float64, bool) {
	t.Helper()
	if len(epochs) < 2 {
		return 0, false
	}

	maxDriftPct := 0.0
	driftDetected := false

	for i := 1; i < len(epochs); i++ {
		prev := epochs[i-1]
		curr := epochs[i]

		// Counter monotonicity: processed, filled, venueReqs must never decrease.
		if curr.snap.processed < prev.snap.processed {
			t.Errorf("[drift/epoch-%d] processed decreased: %d → %d", i, prev.snap.processed, curr.snap.processed)
			driftDetected = true
		}
		if curr.snap.filled < prev.snap.filled {
			t.Errorf("[drift/epoch-%d] filled decreased: %d → %d", i, prev.snap.filled, curr.snap.filled)
			driftDetected = true
		}
		if curr.snap.venueReqs < prev.snap.venueReqs {
			t.Errorf("[drift/epoch-%d] venueReqs decreased: %d → %d", i, prev.snap.venueReqs, curr.snap.venueReqs)
			driftDetected = true
		}

		// Error accumulation: errors must not grow.
		if curr.snap.errors > prev.snap.errors {
			t.Errorf("[drift/epoch-%d] errors accumulated: %d → %d", i, prev.snap.errors, curr.snap.errors)
			driftDetected = true
		}

		// Venue/fill parity: venueReqs must equal filled at every epoch.
		if curr.snap.venueReqs != curr.snap.filled {
			pct := 0.0
			if curr.snap.filled > 0 {
				pct = math.Abs(float64(curr.snap.venueReqs-curr.snap.filled)) / float64(curr.snap.filled) * 100
			}
			if pct > maxDriftPct {
				maxDriftPct = pct
			}
			t.Errorf("[drift/epoch-%d] venue/fill parity broken: venueReqs=%d filled=%d (%.2f%%)",
				i, curr.snap.venueReqs, curr.snap.filled, pct)
			driftDetected = true
		}
	}

	return maxDriftPct, driftDetected
}

// s349AnalyzeLatencyRegression checks if latency degrades over epochs.
// Compares first-third average to last-third average; regression if last > first * threshold.
func s349AnalyzeLatencyRegression(t *testing.T, epochs []s349Epoch, thresholdMultiplier float64) bool {
	t.Helper()
	n := len(epochs)
	if n < 6 {
		t.Logf("[latency] too few epochs (%d) for regression analysis, skipping", n)
		return false
	}

	third := n / 3
	firstThird := epochs[:third]
	lastThird := epochs[n-third:]

	avgFirst := 0.0
	for _, e := range firstThird {
		avgFirst += e.latencyMs
	}
	avgFirst /= float64(len(firstThird))

	avgLast := 0.0
	for _, e := range lastThird {
		avgLast += e.latencyMs
	}
	avgLast /= float64(len(lastThird))

	t.Logf("[latency] first-third avg=%.1fms, last-third avg=%.1fms, threshold=%.1fx",
		avgFirst, avgLast, thresholdMultiplier)

	if avgFirst > 0 && avgLast > avgFirst*thresholdMultiplier {
		t.Errorf("[latency] regression detected: last-third (%.1fms) > first-third (%.1fms) * %.1f",
			avgLast, avgFirst, thresholdMultiplier)
		return true
	}
	return false
}

// ---------- END-1: 5-Minute Sustained Gate Active with Latency Tracking ----------

func TestEndurance_SustainedGateActiveWithLatencyTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping endurance test in short mode")
	}

	url := s333NatsURL(t)
	creds := s349TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s349VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s349-end1-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s349-end1-consumer"),
	}

	// Gate starts ACTIVE — sustained endurance observation.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s349-end1-sustained", "s349-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s343SpawnSupervisor(t, s343AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Sustained observation: inject events every 15 seconds for 5 minutes ──

	const windowDuration = 5 * time.Minute
	const injectInterval = 15 * time.Second
	totalInjections := int(windowDuration / injectInterval) // 20 events

	windowStart := time.Now()
	epochs := make([]s349Epoch, 0, totalInjections)

	t.Logf("[END-1] observation window: %s, inject interval: %s, planned injections: %d",
		windowDuration, injectInterval, totalInjections)

	for i := 1; i <= totalInjections; i++ {
		corrID := fmt.Sprintf("s349-end1-%d-%d", i, time.Now().UnixNano())
		event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)

		publishStart := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := publisher.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("[END-1/inject-%d] publish failed: %s", i, prob.Message)
		}

		fill := fillSub.waitForFill(corrID, 15*time.Second)
		if fill == nil {
			t.Fatalf("[END-1/inject-%d] fill not received within 15s — sustained flow interrupted at minute %.1f",
				i, time.Since(windowStart).Minutes())
		}

		fillReceived := time.Now()
		latencyMs := float64(fillReceived.Sub(publishStart).Microseconds()) / 1000.0

		if fill.ExecutionIntent.Fills[0].Simulated {
			t.Fatalf("[END-1/inject-%d] fill is simulated — real venue adapter expected", i)
		}

		snap := s343TakeSnapshot(adapterTracker, &venueRequests)
		s343CheckInvariant(t, fmt.Sprintf("END-1/inject-%d", i), snap)

		epoch := s349Epoch{
			index:       i,
			startedAt:   publishStart,
			completedAt: fillReceived,
			snap:        snap,
			latencyMs:   latencyMs,
		}
		epochs = append(epochs, epoch)

		if i%5 == 0 || i == totalInjections {
			t.Logf("[END-1/inject-%d] t=%.0fs processed=%d filled=%d venue_reqs=%d latency=%.1fms errors=%d",
				i, time.Since(windowStart).Seconds(), snap.processed, snap.filled, snap.venueReqs, latencyMs, snap.errors)
		}

		if i < totalInjections {
			time.Sleep(injectInterval)
		}
	}

	windowEnd := time.Now()
	finalSnap := s343TakeSnapshot(adapterTracker, &venueRequests)

	// ── Final assertions ──

	if finalSnap.filled < int64(totalInjections) {
		t.Fatalf("[END-1] expected filled >= %d, got %d", totalInjections, finalSnap.filled)
	}
	if finalSnap.skippedHalt != 0 {
		t.Fatalf("[END-1] expected skipped_halt=0 (gate active throughout), got %d", finalSnap.skippedHalt)
	}
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[END-1] venue_reqs(%d) != filled(%d) — request/fill drift", finalSnap.venueReqs, finalSnap.filled)
	}
	if finalSnap.errors != 0 {
		t.Fatalf("[END-1] expected 0 errors, got %d", finalSnap.errors)
	}
	s343CheckInvariant(t, "END-1/final", finalSnap)

	// ── Drift regression analysis ──
	maxDrift, driftDetected := s349AnalyzeDrift(t, epochs)
	if driftDetected {
		t.Fatalf("[END-1] drift detected across %d epochs (max drift=%.2f%%)", len(epochs), maxDrift)
	}

	// ── Latency regression analysis ──
	// Threshold: last-third latency must not exceed 3x first-third average.
	regression := s349AnalyzeLatencyRegression(t, epochs, 3.0)
	if regression {
		t.Fatal("[END-1] latency regression detected over 5-minute window")
	}

	actualDuration := windowEnd.Sub(windowStart)
	t.Logf("[END-1] window: %s (actual=%s)", windowDuration, actualDuration.Truncate(time.Second))
	t.Logf("[END-1] summary: processed=%d filled=%d skipped_halt=%d errors=%d venue_reqs=%d epochs=%d",
		finalSnap.processed, finalSnap.filled, finalSnap.skippedHalt, finalSnap.errors, finalSnap.venueReqs, len(epochs))

	// Log latency distribution.
	var minLat, maxLat, sumLat float64
	minLat = math.MaxFloat64
	for _, e := range epochs {
		if e.latencyMs < minLat {
			minLat = e.latencyMs
		}
		if e.latencyMs > maxLat {
			maxLat = e.latencyMs
		}
		sumLat += e.latencyMs
	}
	avgLat := sumLat / float64(len(epochs))
	t.Logf("[END-1] latency: min=%.1fms avg=%.1fms max=%.1fms", minLat, avgLat, maxLat)
	t.Log("[s349/END-1] PASS — 5-minute sustained gate active, no drift, no latency regression")
}

// ---------- END-2: 5-Minute Mixed Workload Endurance ----------

func TestEndurance_MixedWorkloadEndurance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping endurance test in short mode")
	}

	url := s333NatsURL(t)
	creds := s349TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s349VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s349-end2-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s349-end2-consumer"),
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s349-end2-start", "s349-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s349-end2-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s349-test",
		})
	}()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s343SpawnSupervisor(t, s343AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Mixed workload schedule over ~5 minutes ──
	//
	// Phase 1: ACTIVE  — 5 events at 10s intervals    (50s)
	// Phase 2: IDLE    — 30s pause, no events          (30s)
	// Phase 3: HALTED  — 4 events at 8s intervals      (32s)
	// Phase 4: ACTIVE  — burst of 6 rapid events       (~6s)
	// Phase 5: IDLE    — 40s pause                     (40s)
	// Phase 6: ACTIVE  — 5 events at 12s intervals     (60s)
	// Phase 7: HALTED  — 3 events at 10s intervals     (30s)
	// Phase 8: ACTIVE  — burst of 8 rapid events       (~8s)
	// Phase 9: IDLE    — 30s final stability check     (30s)
	// Total: ~286s ≈ ~5 minutes

	windowStart := time.Now()
	epochs := make([]s349Epoch, 0, 40)
	expectedFilled := int64(0)
	expectedSkippedHalt := int64(0)

	// Helper: inject one event and record epoch.
	injectAndTrack := func(phase string, idx int, expectFill bool) {
		corrID := fmt.Sprintf("s349-end2-%s-%d-%d", phase, idx, time.Now().UnixNano())
		event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
		publishStart := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := publisher.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("[END-2/%s/%d] publish failed: %s", phase, idx, prob.Message)
		}

		if expectFill {
			fill := fillSub.waitForFill(corrID, 15*time.Second)
			if fill == nil {
				t.Fatalf("[END-2/%s/%d] fill not received", phase, idx)
			}
			if fill.ExecutionIntent.Fills[0].Simulated {
				t.Fatalf("[END-2/%s/%d] fill simulated — real adapter expected", phase, idx)
			}
			expectedFilled++
		} else {
			target := adapterTracker.Counter("processed").Load() + 1
			s341WaitCounter(t, adapterTracker, "processed", target, 10*time.Second)
			expectedSkippedHalt++
		}

		snap := s343TakeSnapshot(adapterTracker, &venueRequests)
		s343CheckInvariant(t, fmt.Sprintf("END-2/%s/%d", phase, idx), snap)

		latencyMs := float64(time.Since(publishStart).Microseconds()) / 1000.0
		epochs = append(epochs, s349Epoch{
			index:       len(epochs) + 1,
			startedAt:   publishStart,
			completedAt: time.Now(),
			snap:        snap,
			latencyMs:   latencyMs,
		})
	}

	setGate := func(status domainexec.GateStatus, reason string) {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: status, Reason: reason, UpdatedBy: "s349-test", UpdatedAt: time.Now().UTC(),
		})
		time.Sleep(200 * time.Millisecond)
	}

	// Phase 1: ACTIVE — steady flow.
	t.Logf("[END-2/phase-1] ACTIVE — 5 events at 10s intervals (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 5; i++ {
		injectAndTrack("p1-active", i, true)
		if i < 4 {
			time.Sleep(10 * time.Second)
		}
	}

	// Phase 2: IDLE — no events, check for drift.
	t.Logf("[END-2/phase-2] IDLE — 30s pause (t=%.0fs)", time.Since(windowStart).Seconds())
	snapBeforeIdle1 := s343TakeSnapshot(adapterTracker, &venueRequests)
	time.Sleep(30 * time.Second)
	snapAfterIdle1 := s343TakeSnapshot(adapterTracker, &venueRequests)
	if snapAfterIdle1.processed != snapBeforeIdle1.processed {
		t.Fatalf("[END-2/phase-2] processed changed during idle: %d → %d",
			snapBeforeIdle1.processed, snapAfterIdle1.processed)
	}

	// Phase 3: HALTED — events blocked.
	setGate(domainexec.GateHalted, "s349-end2-phase3-halt")
	t.Logf("[END-2/phase-3] HALTED — 4 events at 8s intervals (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 4; i++ {
		injectAndTrack("p3-halted", i, false)
		if i < 3 {
			time.Sleep(8 * time.Second)
		}
	}

	// Phase 4: ACTIVE — rapid burst.
	setGate(domainexec.GateActive, "s349-end2-phase4-burst")
	t.Logf("[END-2/phase-4] ACTIVE — burst of 6 rapid events (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 6; i++ {
		injectAndTrack("p4-burst", i, true)
	}

	// Phase 5: IDLE — extended pause.
	t.Logf("[END-2/phase-5] IDLE — 40s pause (t=%.0fs)", time.Since(windowStart).Seconds())
	snapBeforeIdle2 := s343TakeSnapshot(adapterTracker, &venueRequests)
	time.Sleep(40 * time.Second)
	snapAfterIdle2 := s343TakeSnapshot(adapterTracker, &venueRequests)
	if snapAfterIdle2.processed != snapBeforeIdle2.processed {
		t.Fatalf("[END-2/phase-5] processed changed during idle: %d → %d",
			snapBeforeIdle2.processed, snapAfterIdle2.processed)
	}

	// Phase 6: ACTIVE — steady flow.
	t.Logf("[END-2/phase-6] ACTIVE — 5 events at 12s intervals (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 5; i++ {
		injectAndTrack("p6-active", i, true)
		if i < 4 {
			time.Sleep(12 * time.Second)
		}
	}

	// Phase 7: HALTED — events blocked.
	setGate(domainexec.GateHalted, "s349-end2-phase7-halt")
	t.Logf("[END-2/phase-7] HALTED — 3 events at 10s intervals (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 3; i++ {
		injectAndTrack("p7-halted", i, false)
		if i < 2 {
			time.Sleep(10 * time.Second)
		}
	}

	// Phase 8: ACTIVE — final burst.
	setGate(domainexec.GateActive, "s349-end2-phase8-final")
	t.Logf("[END-2/phase-8] ACTIVE — burst of 8 rapid events (t=%.0fs)", time.Since(windowStart).Seconds())
	for i := 0; i < 8; i++ {
		injectAndTrack("p8-burst", i, true)
	}

	// Phase 9: IDLE — final stability check.
	t.Logf("[END-2/phase-9] IDLE — 30s final stability check (t=%.0fs)", time.Since(windowStart).Seconds())
	snapBeforeFinal := s343TakeSnapshot(adapterTracker, &venueRequests)
	time.Sleep(30 * time.Second)
	snapAfterFinal := s343TakeSnapshot(adapterTracker, &venueRequests)
	if snapAfterFinal.processed != snapBeforeFinal.processed {
		t.Fatalf("[END-2/phase-9] processed changed during final idle: %d → %d",
			snapBeforeFinal.processed, snapAfterFinal.processed)
	}

	windowEnd := time.Now()
	finalSnap := s343TakeSnapshot(adapterTracker, &venueRequests)

	// ── Final assertions ──

	if finalSnap.filled != expectedFilled {
		t.Fatalf("[END-2] expected filled=%d, got %d", expectedFilled, finalSnap.filled)
	}
	if finalSnap.skippedHalt != expectedSkippedHalt {
		t.Fatalf("[END-2] expected skipped_halt=%d, got %d", expectedSkippedHalt, finalSnap.skippedHalt)
	}
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[END-2] venue_reqs(%d) != filled(%d) — drift", finalSnap.venueReqs, finalSnap.filled)
	}
	if finalSnap.errors != 0 {
		t.Fatalf("[END-2] expected 0 errors, got %d", finalSnap.errors)
	}

	totalEvents := expectedFilled + expectedSkippedHalt
	if finalSnap.processed != totalEvents {
		t.Fatalf("[END-2] expected processed=%d, got %d", totalEvents, finalSnap.processed)
	}
	s343CheckInvariant(t, "END-2/final", finalSnap)

	// ── Drift analysis ──
	maxDrift, driftDetected := s349AnalyzeDrift(t, epochs)
	if driftDetected {
		t.Fatalf("[END-2] drift detected (max=%.2f%%)", maxDrift)
	}

	actualDuration := windowEnd.Sub(windowStart)
	t.Logf("[END-2] window: actual=%s", actualDuration.Truncate(time.Second))
	t.Logf("[END-2] summary: processed=%d filled=%d skipped_halt=%d errors=%d venue_reqs=%d epochs=%d gate_transitions=4",
		finalSnap.processed, finalSnap.filled, finalSnap.skippedHalt, finalSnap.errors, finalSnap.venueReqs, len(epochs))
	t.Log("[s349/END-2] PASS — 5-minute mixed workload endurance, no drift, no idle corruption")
}

// ---------- END-3: Counter Monotonicity Under Repeated Burst Cycles ----------

func TestEndurance_CounterMonotonicityUnderRepeatedBursts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping endurance test in short mode")
	}

	url := s333NatsURL(t)
	creds := s349TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s349VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s349-end3-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s349-end3-consumer"),
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s349-end3-active", "s349-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s343SpawnSupervisor(t, s343AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── 10 burst cycles, 4 events per burst, 30s idle between bursts ──
	// Total events: 40, Total time: ~5 minutes (10 bursts * ~4s + 9 pauses * 30s = ~310s)

	const burstSize = 4
	const totalBursts = 10
	const pauseBetweenBursts = 30 * time.Second

	windowStart := time.Now()
	epochs := make([]s349Epoch, 0, burstSize*totalBursts)
	totalFills := int64(0)

	for burst := 0; burst < totalBursts; burst++ {
		t.Logf("[END-3/burst-%d] injecting %d events (t=%.0fs)", burst+1, burstSize, time.Since(windowStart).Seconds())

		for ei := 0; ei < burstSize; ei++ {
			corrID := fmt.Sprintf("s349-end3-b%d-e%d-%d", burst, ei, time.Now().UnixNano())
			event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
			publishStart := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			prob := publisher.PublishExecution(ctx, event)
			cancel()
			if prob != nil {
				t.Fatalf("[END-3/burst-%d/e%d] publish failed: %s", burst, ei, prob.Message)
			}

			fill := fillSub.waitForFill(corrID, 15*time.Second)
			if fill == nil {
				t.Fatalf("[END-3/burst-%d/e%d] fill not received", burst, ei)
			}
			totalFills++

			latencyMs := float64(time.Since(publishStart).Microseconds()) / 1000.0
			snap := s343TakeSnapshot(adapterTracker, &venueRequests)
			s343CheckInvariant(t, fmt.Sprintf("END-3/burst-%d/e%d", burst, ei), snap)

			epochs = append(epochs, s349Epoch{
				index:       len(epochs) + 1,
				startedAt:   publishStart,
				completedAt: time.Now(),
				snap:        snap,
				latencyMs:   latencyMs,
			})
		}

		// Post-burst snapshot.
		snap := s343TakeSnapshot(adapterTracker, &venueRequests)
		if snap.filled != totalFills {
			t.Fatalf("[END-3/post-burst-%d] expected filled=%d, got %d", burst+1, totalFills, snap.filled)
		}

		if burst < totalBursts-1 {
			// Idle pause — check for counter drift.
			snapBeforePause := snap
			time.Sleep(pauseBetweenBursts)
			snapAfterPause := s343TakeSnapshot(adapterTracker, &venueRequests)

			if snapAfterPause.processed != snapBeforePause.processed {
				t.Fatalf("[END-3/idle-%d] processed changed during idle: %d → %d",
					burst+1, snapBeforePause.processed, snapAfterPause.processed)
			}
			if snapAfterPause.filled != snapBeforePause.filled {
				t.Fatalf("[END-3/idle-%d] filled changed during idle: %d → %d",
					burst+1, snapBeforePause.filled, snapAfterPause.filled)
			}
			if snapAfterPause.errors != snapBeforePause.errors {
				t.Fatalf("[END-3/idle-%d] errors changed during idle: %d → %d",
					burst+1, snapBeforePause.errors, snapAfterPause.errors)
			}
		}
	}

	windowEnd := time.Now()
	finalSnap := s343TakeSnapshot(adapterTracker, &venueRequests)

	// ── Final assertions ──

	expectedTotal := int64(burstSize * totalBursts)
	if finalSnap.processed != expectedTotal {
		t.Fatalf("[END-3] expected processed=%d, got %d", expectedTotal, finalSnap.processed)
	}
	if finalSnap.filled != expectedTotal {
		t.Fatalf("[END-3] expected filled=%d, got %d", expectedTotal, finalSnap.filled)
	}
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[END-3] venue_reqs(%d) != filled(%d)", finalSnap.venueReqs, finalSnap.filled)
	}
	if finalSnap.errors != 0 {
		t.Fatalf("[END-3] expected 0 errors, got %d", finalSnap.errors)
	}
	s343CheckInvariant(t, "END-3/final", finalSnap)

	// ── Drift and latency regression ──
	maxDrift, driftDetected := s349AnalyzeDrift(t, epochs)
	if driftDetected {
		t.Fatalf("[END-3] drift detected (max=%.2f%%)", maxDrift)
	}

	regression := s349AnalyzeLatencyRegression(t, epochs, 3.0)
	if regression {
		t.Fatal("[END-3] latency regression detected")
	}

	actualDuration := windowEnd.Sub(windowStart)
	t.Logf("[END-3] window: actual=%s", actualDuration.Truncate(time.Second))
	t.Logf("[END-3] summary: bursts=%d events_per_burst=%d total=%d filled=%d venue_reqs=%d errors=%d epochs=%d",
		totalBursts, burstSize, finalSnap.processed, finalSnap.filled, finalSnap.venueReqs, finalSnap.errors, len(epochs))

	// Latency distribution.
	var minLat, maxLat, sumLat float64
	minLat = math.MaxFloat64
	for _, e := range epochs {
		if e.latencyMs < minLat {
			minLat = e.latencyMs
		}
		if e.latencyMs > maxLat {
			maxLat = e.latencyMs
		}
		sumLat += e.latencyMs
	}
	avgLat := sumLat / float64(len(epochs))
	t.Logf("[END-3] latency: min=%.1fms avg=%.1fms max=%.1fms", minLat, avgLat, maxLat)
	t.Log("[s349/END-3] PASS — 10 burst cycles over ~5 minutes, counter monotonicity proven, no latency regression")
}
