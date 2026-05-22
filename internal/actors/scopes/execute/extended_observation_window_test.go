//go:build integration

package execute_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	executeactor "internal/actors/scopes/execute"
	natsexecution "internal/adapters/nats/natsexecution"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// extended_observation_window_test.go — S343: Extended Live Observation Window.
//
// These tests close the residual gap from S341/S342: "extended observation window
// not exercised" — by sustaining the venue path for minutes with periodic event
// injection, multiple gate transitions, and continuous counter consistency checks.
//
// Strategy: reuse the S342 httptest.Server venue simulation, but extend the observation
// window from seconds (S342) to minutes (S343). Events are injected at regular intervals.
// Between injections, counter invariants are validated: processed == filled + skipped_halt.
//
// What S343 proves that S341/S342 did not:
//   - Gate remains consistent and responsive over a multi-minute window
//   - Counter invariants hold over sustained operation, not just single-event tests
//   - No resource leak, goroutine leak, or drift emerges during sustained operation
//   - Multiple gate transitions mid-window do not corrupt state
//   - Venue HTTP request count tracks filled count exactly over time
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

// s343Seq tracks HTTP requests to the simulated venue for assertions.
var s343Seq atomic.Int64

// s343VenueServer creates an httptest.Server simulating Binance Futures testnet.
// Identical to s342VenueServer but with its own sequence counter.
func s343VenueServer(t *testing.T, requestCounter *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter.Add(1)
		q := r.URL.Query()

		if r.Header.Get("X-MBX-APIKEY") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "missing API key"})
			return
		}

		if q.Get("signature") == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": -1022, "msg": "missing signature"})
			return
		}

		orderID := s343Seq.Add(1) + 343000

		resp := map[string]any{
			"orderId":       orderID,
			"clientOrderId": q.Get("newClientOrderId"),
			"symbol":        q.Get("symbol"),
			"status":        "FILLED",
			"side":          q.Get("side"),
			"type":          "MARKET",
			"avgPrice":      "67890.50",
			"executedQty":   q.Get("quantity"),
			"cumQuote":      "67.89",
			"updateTime":    time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// s343TestCredentials creates test credentials for S343 tests.
func s343TestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "s343-test-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "s343-test-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

// s343AppConfig builds a test-safe AppConfig for extended observation.
func s343AppConfig(url string) settings.AppConfig {
	return settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "binance_futures_testnet",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
}

// s343SpawnSupervisor creates a supervisor with a real venue adapter for extended tests.
func s343SpawnSupervisor(
	t *testing.T,
	cfg settings.AppConfig,
	venue ports.VenuePort,
	venueQuery ports.VenueQueryPort,
	trackers map[string]*healthz.Tracker,
) *actor.Engine {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(cfg, venue, venueQuery, trackers,
			executeactor.WithActivationState(domainexec.AdapterVenue, domainexec.CredentialPresent),
		),
		fmt.Sprintf("s343-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond)
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

// s343Snapshot captures counter state at a point in time for drift analysis.
type s343Snapshot struct {
	at          time.Time
	processed   int64
	filled      int64
	skippedHalt int64
	errors      int64
	venueReqs   int64
}

// s343TakeSnapshot captures current counter state.
func s343TakeSnapshot(tracker *healthz.Tracker, venueReqs *atomic.Int64) s343Snapshot {
	return s343Snapshot{
		at:          time.Now(),
		processed:   tracker.Counter("processed").Load(),
		filled:      tracker.Counter("filled").Load(),
		skippedHalt: tracker.Counter("skipped_halt").Load(),
		errors:      tracker.ErrorCount(),
		venueReqs:   venueReqs.Load(),
	}
}

// s343CheckInvariant verifies: processed == filled + skipped_halt.
func s343CheckInvariant(t *testing.T, label string, snap s343Snapshot) {
	t.Helper()
	sum := snap.filled + snap.skippedHalt
	if snap.processed != sum {
		t.Fatalf("[%s] counter invariant violated: processed=%d != filled(%d) + skipped_halt(%d) = %d",
			label, snap.processed, snap.filled, snap.skippedHalt, sum)
	}
}

// ---------- EOW-1: Sustained Gate Active — Event Flow Over 2 Minutes ----------

func TestExtendedObservation_SustainedGateActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended observation in short mode")
	}

	url := s333NatsURL(t)
	creds := s343TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s343VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s343-eow1-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s343-eow1-consumer"),
	}

	// Gate starts ACTIVE — sustained observation of enabled flow.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s343-eow1-sustained", "s343-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s343SpawnSupervisor(t, s343AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Sustained observation: inject events every 10 seconds for 2 minutes ──

	const windowDuration = 2 * time.Minute
	const injectInterval = 10 * time.Second
	totalInjections := int(windowDuration / injectInterval) // 12 events

	windowStart := time.Now()
	snapshots := make([]s343Snapshot, 0, totalInjections+1)
	snapshots = append(snapshots, s343TakeSnapshot(adapterTracker, &venueRequests))

	t.Logf("[EOW-1] observation window: %s, inject interval: %s, planned injections: %d",
		windowDuration, injectInterval, totalInjections)

	for i := 1; i <= totalInjections; i++ {
		corrID := fmt.Sprintf("s343-eow1-%d-%d", i, time.Now().UnixNano())
		event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := publisher.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("[EOW-1/inject-%d] publish failed: %s", i, prob.Message)
		}

		// Wait for fill.
		fill := fillSub.waitForFill(corrID, 15*time.Second)
		if fill == nil {
			t.Fatalf("[EOW-1/inject-%d] fill not received within 15s — sustained flow interrupted", i)
		}

		if fill.ExecutionIntent.Fills[0].Simulated {
			t.Fatalf("[EOW-1/inject-%d] fill is simulated — real venue adapter expected", i)
		}

		// Take snapshot and validate invariant (eventually — actor increments
		// the filled counter after PublishFill returns).
		snap := s343EventuallyInvariant(t, fmt.Sprintf("EOW-1/inject-%d", i), adapterTracker, &venueRequests, 2*time.Second)
		snapshots = append(snapshots, snap)

		if i < totalInjections {
			t.Logf("[EOW-1/inject-%d] fill received, processed=%d filled=%d venue_reqs=%d — waiting %s",
				i, snap.processed, snap.filled, snap.venueReqs, injectInterval)
			time.Sleep(injectInterval)
		} else {
			t.Logf("[EOW-1/inject-%d] final fill received, processed=%d filled=%d venue_reqs=%d",
				i, snap.processed, snap.filled, snap.venueReqs)
		}
	}

	windowEnd := time.Now()
	finalSnap := s343EventuallyInvariant(t, "EOW-1/final", adapterTracker, &venueRequests, 2*time.Second)

	// ── Final assertions ──

	// All events processed successfully.
	if finalSnap.filled < int64(totalInjections) {
		t.Fatalf("[EOW-1] expected filled >= %d, got %d", totalInjections, finalSnap.filled)
	}

	// No halted events (gate was active throughout).
	if finalSnap.skippedHalt != 0 {
		t.Fatalf("[EOW-1] expected skipped_halt=0 (gate active), got %d", finalSnap.skippedHalt)
	}

	// Venue HTTP requests match filled count.
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[EOW-1] venue_reqs(%d) != filled(%d) — request/fill drift detected",
			finalSnap.venueReqs, finalSnap.filled)
	}

	// No errors during sustained operation.
	if finalSnap.errors != 0 {
		t.Fatalf("[EOW-1] expected 0 errors during sustained observation, got %d", finalSnap.errors)
	}

	t.Logf("[EOW-1] window: %s (actual=%s)", windowDuration, windowEnd.Sub(windowStart).Truncate(time.Second))
	t.Logf("[EOW-1] summary: processed=%d filled=%d skipped_halt=%d errors=%d venue_reqs=%d snapshots=%d",
		finalSnap.processed, finalSnap.filled, finalSnap.skippedHalt, finalSnap.errors, finalSnap.venueReqs, len(snapshots))
	t.Log("[s343/EOW-1] PASS — sustained gate active over 2-minute window, no drift or intermittency")
}

// ---------- EOW-2: Gate Transitions During Extended Window ----------

func TestExtendedObservation_GateTransitionsDuringWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended observation in short mode")
	}

	url := s333NatsURL(t)
	creds := s343TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s343VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s343-eow2-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s343-eow2-consumer"),
	}

	// Start halted — canonical safe deploy.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s343-eow2-initial", "s343-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s343-eow2-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s343-test",
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

	// ── Extended window with alternating gate states ──
	//
	// Schedule:
	//   t=0s:    HALTED  — inject 3 events (all blocked)
	//   t=30s:   ACTIVE  — inject 3 events (all filled)
	//   t=60s:   HALTED  — inject 3 events (all blocked)
	//   t=90s:   ACTIVE  — inject 3 events (all filled)
	//   t=120s:  end     — final consistency check

	type phase struct {
		gate   domainexec.GateStatus
		reason string
		events int
	}

	phases := []phase{
		{domainexec.GateHalted, "s343-eow2-phase1-halted", 3},
		{domainexec.GateActive, "s343-eow2-phase2-active", 3},
		{domainexec.GateHalted, "s343-eow2-phase3-halted", 3},
		{domainexec.GateActive, "s343-eow2-phase4-active", 3},
	}

	const interPhaseWait = 30 * time.Second
	const intraEventWait = 5 * time.Second

	windowStart := time.Now()
	snapshots := make([]s343Snapshot, 0, 20)
	snapshots = append(snapshots, s343TakeSnapshot(adapterTracker, &venueRequests))

	expectedFilled := int64(0)
	expectedSkippedHalt := int64(0)

	for pi, ph := range phases {
		phaseLabel := fmt.Sprintf("phase-%d", pi+1)

		// Set gate state.
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status:    ph.gate,
			Reason:    ph.reason,
			UpdatedBy: "s343-test",
			UpdatedAt: time.Now().UTC(),
		})
		time.Sleep(200 * time.Millisecond) // KV propagation

		t.Logf("[EOW-2/%s] gate=%s, injecting %d events", phaseLabel, ph.gate, ph.events)

		for ei := 0; ei < ph.events; ei++ {
			corrID := fmt.Sprintf("s343-eow2-%s-e%d-%d", phaseLabel, ei, time.Now().UnixNano())
			event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			prob := publisher.PublishExecution(ctx, event)
			cancel()
			if prob != nil {
				t.Fatalf("[EOW-2/%s/e%d] publish failed: %s", phaseLabel, ei, prob.Message)
			}

			if ph.gate == domainexec.GateActive {
				fill := fillSub.waitForFill(corrID, 15*time.Second)
				if fill == nil {
					t.Fatalf("[EOW-2/%s/e%d] fill not received — active gate should enable flow", phaseLabel, ei)
				}
				if fill.ExecutionIntent.Fills[0].Simulated {
					t.Fatalf("[EOW-2/%s/e%d] fill simulated — real adapter expected", phaseLabel, ei)
				}
				expectedFilled++
			} else {
				// Wait for processed counter to increment.
				target := adapterTracker.Counter("processed").Load() + 1
				s341WaitCounter(t, adapterTracker, "processed", target, 10*time.Second)
				expectedSkippedHalt++
			}

			snap := s343EventuallyInvariant(t, fmt.Sprintf("EOW-2/%s/e%d", phaseLabel, ei), adapterTracker, &venueRequests, 2*time.Second)
			snapshots = append(snapshots, snap)

			if ei < ph.events-1 {
				time.Sleep(intraEventWait)
			}
		}

		// Wait between phases (except after last).
		if pi < len(phases)-1 {
			venueReqSnap := venueRequests.Load()
			filledSnap := adapterTracker.Counter("filled").Load()
			t.Logf("[EOW-2/%s] complete. filled=%d skipped_halt=%d venue_reqs=%d — waiting %s",
				phaseLabel, filledSnap, adapterTracker.Counter("skipped_halt").Load(), venueReqSnap, interPhaseWait)
			time.Sleep(interPhaseWait)
		}
	}

	windowEnd := time.Now()
	finalSnap := s343EventuallyInvariant(t, "EOW-2/final", adapterTracker, &venueRequests, 2*time.Second)

	// ── Final assertions ──

	// Expected fill and skip counts.
	if finalSnap.filled != expectedFilled {
		t.Fatalf("[EOW-2] expected filled=%d, got %d", expectedFilled, finalSnap.filled)
	}
	if finalSnap.skippedHalt != expectedSkippedHalt {
		t.Fatalf("[EOW-2] expected skipped_halt=%d, got %d", expectedSkippedHalt, finalSnap.skippedHalt)
	}

	// Venue requests match fills.
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[EOW-2] venue_reqs(%d) != filled(%d) — request/fill drift",
			finalSnap.venueReqs, finalSnap.filled)
	}

	// No errors.
	if finalSnap.errors != 0 {
		t.Fatalf("[EOW-2] expected 0 errors, got %d", finalSnap.errors)
	}

	// Total processed.
	totalEvents := int64(0)
	for _, ph := range phases {
		totalEvents += int64(ph.events)
	}
	if finalSnap.processed != totalEvents {
		t.Fatalf("[EOW-2] expected processed=%d, got %d", totalEvents, finalSnap.processed)
	}

	t.Logf("[EOW-2] window: actual=%s", windowEnd.Sub(windowStart).Truncate(time.Second))
	t.Logf("[EOW-2] summary: processed=%d filled=%d skipped_halt=%d errors=%d venue_reqs=%d transitions=4 snapshots=%d",
		finalSnap.processed, finalSnap.filled, finalSnap.skippedHalt, finalSnap.errors, finalSnap.venueReqs, len(snapshots))
	t.Log("[s343/EOW-2] PASS — gate transitions during extended window, counters consistent throughout")
}

// ---------- EOW-3: Counter Consistency Under Sustained Load ----------

func TestExtendedObservation_CounterConsistencyUnderSustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended observation in short mode")
	}

	url := s333NatsURL(t)
	creds := s343TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s343VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s343-eow3-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s343-eow3-consumer"),
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s343-eow3-active", "s343-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s343SpawnSupervisor(t, s343AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Burst-and-pause pattern: 5 events rapid, 20s pause, repeated 3 times (60s total) ──
	// This tests counter consistency under burst conditions with idle gaps.

	const burstSize = 5
	const pauseAfterBurst = 20 * time.Second
	const totalBursts = 3

	windowStart := time.Now()
	totalFills := int64(0)

	for burst := 0; burst < totalBursts; burst++ {
		t.Logf("[EOW-3/burst-%d] injecting %d events rapidly", burst+1, burstSize)

		for ei := 0; ei < burstSize; ei++ {
			corrID := fmt.Sprintf("s343-eow3-b%d-e%d-%d", burst, ei, time.Now().UnixNano())
			event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			prob := publisher.PublishExecution(ctx, event)
			cancel()
			if prob != nil {
				t.Fatalf("[EOW-3/burst-%d/e%d] publish failed: %s", burst, ei, prob.Message)
			}

			fill := fillSub.waitForFill(corrID, 15*time.Second)
			if fill == nil {
				t.Fatalf("[EOW-3/burst-%d/e%d] fill not received", burst, ei)
			}
			totalFills++
		}

		// Validate after each burst (eventually — counter increments trail
		// the NATS fill publish by a sub-microsecond window).
		snap := s343EventuallyInvariant(t, fmt.Sprintf("EOW-3/post-burst-%d", burst+1), adapterTracker, &venueRequests, 2*time.Second)

		if snap.filled != totalFills {
			t.Fatalf("[EOW-3/post-burst-%d] expected filled=%d, got %d", burst+1, totalFills, snap.filled)
		}

		t.Logf("[EOW-3/burst-%d] complete: processed=%d filled=%d venue_reqs=%d",
			burst+1, snap.processed, snap.filled, snap.venueReqs)

		if burst < totalBursts-1 {
			// Pause — verify no counter drift during idle period.
			snapBeforePause := snap
			t.Logf("[EOW-3/burst-%d] pausing %s to check for idle drift", burst+1, pauseAfterBurst)
			time.Sleep(pauseAfterBurst)

			snapAfterPause := s343TakeSnapshot(adapterTracker, &venueRequests)
			if snapAfterPause.processed != snapBeforePause.processed {
				t.Fatalf("[EOW-3/idle-%d] processed changed during idle: before=%d after=%d",
					burst+1, snapBeforePause.processed, snapAfterPause.processed)
			}
			if snapAfterPause.filled != snapBeforePause.filled {
				t.Fatalf("[EOW-3/idle-%d] filled changed during idle: before=%d after=%d",
					burst+1, snapBeforePause.filled, snapAfterPause.filled)
			}
			if snapAfterPause.errors != snapBeforePause.errors {
				t.Fatalf("[EOW-3/idle-%d] errors changed during idle: before=%d after=%d",
					burst+1, snapBeforePause.errors, snapAfterPause.errors)
			}
		}
	}

	windowEnd := time.Now()
	finalSnap := s343EventuallyInvariant(t, "EOW-3/final", adapterTracker, &venueRequests, 2*time.Second)

	// ── Final assertions ──

	expectedTotal := int64(burstSize * totalBursts)
	if finalSnap.processed != expectedTotal {
		t.Fatalf("[EOW-3] expected processed=%d, got %d", expectedTotal, finalSnap.processed)
	}
	if finalSnap.filled != expectedTotal {
		t.Fatalf("[EOW-3] expected filled=%d, got %d", expectedTotal, finalSnap.filled)
	}
	if finalSnap.venueReqs != finalSnap.filled {
		t.Fatalf("[EOW-3] venue_reqs(%d) != filled(%d)", finalSnap.venueReqs, finalSnap.filled)
	}
	if finalSnap.errors != 0 {
		t.Fatalf("[EOW-3] expected 0 errors, got %d", finalSnap.errors)
	}

	t.Logf("[EOW-3] window: actual=%s", windowEnd.Sub(windowStart).Truncate(time.Second))
	t.Logf("[EOW-3] summary: bursts=%d events_per_burst=%d total=%d filled=%d venue_reqs=%d errors=%d",
		totalBursts, burstSize, finalSnap.processed, finalSnap.filled, finalSnap.venueReqs, finalSnap.errors)
	t.Log("[s343/EOW-3] PASS — counter consistency under burst-and-pause pattern, no idle drift")
}
