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

// real_venue_activation_verification_test.go — S342: Real Venue Activation Smoke.
//
// These tests close the principal gap from S341: "paper adapter only" — by exercising
// the full activation lifecycle (halted → enabled → halted) with the real
// BinanceFuturesTestnetAdapter instead of the paper simulator.
//
// Strategy: an httptest.Server simulates the Binance Futures testnet API, which lets
// us exercise the REAL adapter code (HTTP signing, query parameter encoding, response
// parsing, fill extraction, error classification) without requiring live testnet
// credentials or network access.
//
// What changes vs S341 tests:
//   - Venue adapter is BinanceFuturesTestnetAdapter (HTTP path), not PaperVenueAdapter
//   - Fills have Simulated=false (real venue behavior)
//   - VenueOrderID is a numeric ID from venue response, not "paper-" prefix
//   - Fill records contain parsed price, quantity, fee from venue JSON response
//   - Supervisor is wired with WithActivationState(AdapterVenue, CredentialPresent)
//   - VenueQuery port is available (adapter implements both VenuePort and VenueQueryPort)
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

// s342Seq tracks HTTP requests to the simulated venue for assertions.
var s342Seq atomic.Int64

// s342VenueServer creates an httptest.Server that simulates the Binance Futures testnet
// POST /fapi/v1/order endpoint, returning FILLED responses with realistic fields.
// The requestCounter is incremented on each request for assertions.
func s342VenueServer(t *testing.T, requestCounter *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter.Add(1)
		q := r.URL.Query()

		// Verify API key header is present (credential wiring proof).
		if r.Header.Get("X-MBX-APIKEY") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "missing API key"})
			return
		}

		// Verify HMAC signature is present (signing pipeline proof).
		if q.Get("signature") == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": -1022, "msg": "missing signature"})
			return
		}

		orderID := s342Seq.Add(1) + 342000

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

// s342RejectionServer creates a venue server that returns HTTP 400 for all orders.
// This simulates venue rejection (e.g. insufficient margin, invalid symbol).
func s342RejectionServer(t *testing.T, requestCounter *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2019,
			"msg":  "Margin is insufficient.",
		})
	}))
}

// s342TestCredentials creates test credentials via environment variables.
func s342TestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "s342-test-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "s342-test-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

// s342AppConfig builds a test-safe AppConfig for real venue activation.
func s342AppConfig(url string) settings.AppConfig {
	return settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "binance_futures_testnet",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
}

// s342SpawnSupervisor creates a supervisor with a real venue adapter wired to the
// httptest server. Unlike s341SpawnSupervisor (which uses PaperVenueAdapter), this
// uses BinanceFuturesTestnetAdapter with WithBaseURL pointing to the test server.
func s342SpawnSupervisor(
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
		fmt.Sprintf("s342-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond)
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

// ---------- RVA-1: Halted Gate Blocks Real Venue Path ----------

func TestRealVenueActivation_HaltedGateBlocksRealVenuePath(t *testing.T) {
	url := s333NatsURL(t)
	creds := s342TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s342VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s342-rva1-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s342-rva1-consumer"),
	}

	// Start with gate HALTED.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s342-rva1-halted", "s342-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s342-rva1-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s342-test",
		})
	}()

	s342SpawnSupervisor(t, s342AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s342-rva1-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for event to reach the actor.
	s341WaitCounter(t, adapterTracker, "processed", 1, 10*time.Second)

	// Verify: event was blocked, not submitted to venue.
	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatalf("[RVA-1] expected skipped_halt >= 1, got %d", adapterTracker.Counter("skipped_halt").Load())
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatalf("[RVA-1] expected filled=0 when halted, got %d", adapterTracker.Counter("filled").Load())
	}
	// Critical: halted gate must prevent any HTTP request to the venue.
	if venueRequests.Load() != 0 {
		t.Fatalf("[RVA-1] expected 0 venue HTTP requests when halted, got %d", venueRequests.Load())
	}

	t.Logf("[RVA-1] processed=%d skipped_halt=%d filled=%d venue_requests=%d",
		adapterTracker.Counter("processed").Load(),
		adapterTracker.Counter("skipped_halt").Load(),
		adapterTracker.Counter("filled").Load(),
		venueRequests.Load())
	t.Log("[s342/RVA-1] PASS — halted gate blocks real venue adapter path (zero HTTP requests)")
}

// ---------- RVA-2: Gate Open Enables Real Venue Flow ----------

func TestRealVenueActivation_GateOpenEnablesRealVenueFlow(t *testing.T) {
	url := s333NatsURL(t)
	creds := s342TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s342VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s342-rva2-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s342-rva2-consumer"),
	}

	// Start with gate ACTIVE.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s342-rva2-enable", "s342-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s342SpawnSupervisor(t, s342AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s342-rva2-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	fill := fillSub.waitForFill(corrID, 10*time.Second)
	if fill == nil {
		t.Fatal("[RVA-2] fill event not received — gate open did not enable real venue flow")
	}

	// Validate: fill is from real adapter (not paper).
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[RVA-2] expected status=filled, got %q", fill.ExecutionIntent.Status)
	}
	if len(fill.ExecutionIntent.Fills) == 0 {
		t.Fatal("[RVA-2] no fill records from real venue adapter")
	}
	fillRecord := fill.ExecutionIntent.Fills[0]
	if fillRecord.Simulated {
		t.Fatal("[RVA-2] real venue fill must have Simulated=false")
	}
	if fillRecord.Price != "67890.50" {
		t.Fatalf("[RVA-2] expected price=67890.50 from venue, got %q", fillRecord.Price)
	}

	// Venue HTTP request was made.
	if venueRequests.Load() < 1 {
		t.Fatalf("[RVA-2] expected >= 1 venue HTTP request, got %d", venueRequests.Load())
	}

	// Correlation preserved.
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("[RVA-2] correlation mismatch: want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}

	// Tracker counters consistent.
	if adapterTracker.Counter("filled").Load() < 1 {
		t.Fatalf("[RVA-2] expected filled >= 1, got %d", adapterTracker.Counter("filled").Load())
	}

	t.Logf("[RVA-2] fill: venue_order_id=%s price=%s simulated=%v venue_requests=%d",
		fill.VenueOrderID, fillRecord.Price, fillRecord.Simulated, venueRequests.Load())
	t.Log("[s342/RVA-2] PASS — gate open enables real venue flow (HTTP adapter, Simulated=false)")
}

// ---------- RVA-3: Runtime Halt Blocks After Enable (Real Venue) ----------

func TestRealVenueActivation_RuntimeHaltBlocksAfterEnable(t *testing.T) {
	url := s333NatsURL(t)
	creds := s342TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s342VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s342-rva3-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s342-rva3-consumer"),
	}

	// Start with gate ACTIVE.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s342-rva3-enable", "s342-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s342-rva3-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s342-test",
		})
	}()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s342SpawnSupervisor(t, s342AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// Phase 1: Verify flow is active with real venue.
	corrID1 := fmt.Sprintf("s342-rva3-live-%d", time.Now().UnixNano())
	event1 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("publish phase 1: %s", prob.Message)
	}

	fill1 := fillSub.waitForFill(corrID1, 10*time.Second)
	if fill1 == nil {
		t.Fatal("[RVA-3/phase-1] fill not received — flow should be active")
	}
	if fill1.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("[RVA-3/phase-1] fill should be non-simulated (real adapter)")
	}
	t.Logf("[RVA-3/phase-1] fill received: venue_order_id=%s", fill1.VenueOrderID)

	filledBeforeHalt := adapterTracker.Counter("filled").Load()
	venueReqsBefore := venueRequests.Load()

	// Phase 2: Halt the gate (runtime transition).
	controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s342-rva3-halt",
		UpdatedBy: "s342-test",
		UpdatedAt: time.Now().UTC(),
	})
	t.Log("[RVA-3/phase-2] gate halted at runtime")
	time.Sleep(200 * time.Millisecond)

	// Phase 3: Publish another event — should be blocked, NO venue HTTP request.
	corrID2 := fmt.Sprintf("s342-rva3-halted-%d", time.Now().UnixNano())
	event2 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("publish phase 3: %s", prob.Message)
	}

	processedBefore := adapterTracker.Counter("processed").Load()
	s341WaitCounter(t, adapterTracker, "processed", processedBefore+1, 10*time.Second)

	// Verify: filled count unchanged.
	filledAfterHalt := adapterTracker.Counter("filled").Load()
	if filledAfterHalt != filledBeforeHalt {
		t.Fatalf("[RVA-3/phase-3] filled changed after halt: before=%d after=%d",
			filledBeforeHalt, filledAfterHalt)
	}

	// Verify: no additional venue requests after halt.
	venueReqsAfter := venueRequests.Load()
	if venueReqsAfter != venueReqsBefore {
		t.Fatalf("[RVA-3/phase-3] venue requests increased after halt: before=%d after=%d",
			venueReqsBefore, venueReqsAfter)
	}

	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatalf("[RVA-3/phase-3] expected skipped_halt >= 1, got %d",
			adapterTracker.Counter("skipped_halt").Load())
	}

	t.Logf("[RVA-3] filled_before=%d filled_after=%d venue_reqs_before=%d venue_reqs_after=%d skipped_halt=%d",
		filledBeforeHalt, filledAfterHalt, venueReqsBefore, venueReqsAfter,
		adapterTracker.Counter("skipped_halt").Load())
	t.Log("[s342/RVA-3] PASS — runtime halt blocks real venue path (zero post-halt HTTP requests)")
}

// ---------- RVA-4: Full Lifecycle with Real Venue (halted → enabled → halted) ----------

func TestRealVenueActivation_FullLifecycle(t *testing.T) {
	url := s333NatsURL(t)
	creds := s342TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s342VenueServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s342-rva4-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s342-rva4-consumer"),
	}

	// Start with gate HALTED — canonical safe deploy posture.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s342-rva4-initial-deploy", "s342-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s342-rva4-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s342-test",
		})
	}()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s342SpawnSupervisor(t, s342AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Phase 1: Halted — event blocked, no venue contact ──

	corrID1 := fmt.Sprintf("s342-rva4-halted-%d", time.Now().UnixNano())
	event1 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("phase 1 publish: %s", prob.Message)
	}

	s341WaitCounter(t, adapterTracker, "processed", 1, 10*time.Second)

	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatal("[RVA-4/phase-1] expected skipped_halt >= 1")
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatal("[RVA-4/phase-1] expected filled=0 while halted")
	}
	if venueRequests.Load() != 0 {
		t.Fatalf("[RVA-4/phase-1] expected 0 venue requests while halted, got %d", venueRequests.Load())
	}
	t.Log("[RVA-4/phase-1] HALTED — event blocked, venue untouched")

	// ── Phase 2: Enable — operator opens gate, real venue fill ──

	controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s342-rva4-smoke-passed",
		UpdatedBy: "s342-operator",
		UpdatedAt: time.Now().UTC(),
	})
	time.Sleep(200 * time.Millisecond)

	corrID2 := fmt.Sprintf("s342-rva4-live-%d", time.Now().UnixNano())
	event2 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("phase 2 publish: %s", prob.Message)
	}

	fill2 := fillSub.waitForFill(corrID2, 10*time.Second)
	if fill2 == nil {
		t.Fatal("[RVA-4/phase-2] fill not received — gate open did not enable real venue flow")
	}
	if fill2.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[RVA-4/phase-2] expected status=filled, got %q", fill2.ExecutionIntent.Status)
	}
	if fill2.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("[RVA-4/phase-2] real venue fill must have Simulated=false")
	}
	if venueRequests.Load() < 1 {
		t.Fatal("[RVA-4/phase-2] expected >= 1 venue HTTP request after enable")
	}
	t.Logf("[RVA-4/phase-2] ENABLED — real venue fill: venue_order_id=%s price=%s",
		fill2.VenueOrderID, fill2.ExecutionIntent.Fills[0].Price)

	filledAfterEnable := adapterTracker.Counter("filled").Load()
	venueReqsAfterEnable := venueRequests.Load()

	// ── Phase 3: Halt — operator halts gate, venue untouched ──

	controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s342-rva4-emergency-halt",
		UpdatedBy: "s342-operator",
		UpdatedAt: time.Now().UTC(),
	})
	time.Sleep(200 * time.Millisecond)

	corrID3 := fmt.Sprintf("s342-rva4-rehalted-%d", time.Now().UnixNano())
	event3 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID3)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event3)
	cancel()
	if prob != nil {
		t.Fatalf("phase 3 publish: %s", prob.Message)
	}

	processedBefore := adapterTracker.Counter("processed").Load()
	s341WaitCounter(t, adapterTracker, "processed", processedBefore+1, 10*time.Second)

	filledAfterHalt := adapterTracker.Counter("filled").Load()
	venueReqsAfterHalt := venueRequests.Load()

	if filledAfterHalt != filledAfterEnable {
		t.Fatalf("[RVA-4/phase-3] filled increased after halt: before=%d after=%d",
			filledAfterEnable, filledAfterHalt)
	}
	if venueReqsAfterHalt != venueReqsAfterEnable {
		t.Fatalf("[RVA-4/phase-3] venue requests increased after halt: before=%d after=%d",
			venueReqsAfterEnable, venueReqsAfterHalt)
	}
	t.Log("[RVA-4/phase-3] HALTED — event blocked, venue untouched after re-halt")

	// ── Summary ──

	t.Logf("[RVA-4] summary: processed=%d filled=%d skipped_halt=%d venue_requests=%d",
		adapterTracker.Counter("processed").Load(),
		adapterTracker.Counter("filled").Load(),
		adapterTracker.Counter("skipped_halt").Load(),
		venueRequests.Load())
	t.Log("[s342/RVA-4] PASS — full lifecycle with real venue: halted → enabled → halted")
}

// ---------- RVA-5: Venue Rejection Does Not Produce Fill ----------

func TestRealVenueActivation_VenueRejectionDoesNotProduceFill(t *testing.T) {
	url := s333NatsURL(t)
	creds := s342TestCredentials(t)

	var venueRequests atomic.Int64
	venueServer := s342RejectionServer(t, &venueRequests)
	defer venueServer.Close()

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(venueServer.URL)

	adapterTracker := healthz.NewTracker("s342-rva5-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s342-rva5-consumer"),
	}

	// Gate ACTIVE — submit reaches the venue, but venue rejects.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s342-rva5-enable", "s342-test")
	defer controlStore.Close()

	s342SpawnSupervisor(t, s342AppConfig(url), adapter, adapter, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s342-rva5-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for event to be processed.
	s341WaitCounter(t, adapterTracker, "processed", 1, 10*time.Second)

	// Give time for any fill to propagate (there shouldn't be one).
	time.Sleep(500 * time.Millisecond)

	// Venue was contacted (gate is active).
	if venueRequests.Load() < 1 {
		t.Fatalf("[RVA-5] expected >= 1 venue request, got %d", venueRequests.Load())
	}

	// No fill produced (venue rejected the order).
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatalf("[RVA-5] expected filled=0 after venue rejection, got %d",
			adapterTracker.Counter("filled").Load())
	}

	// Error was recorded.
	if adapterTracker.ErrorCount() < 1 {
		t.Fatalf("[RVA-5] expected error count >= 1 for venue rejection, got %d",
			adapterTracker.ErrorCount())
	}

	t.Logf("[RVA-5] processed=%d filled=%d errors=%d venue_requests=%d",
		adapterTracker.Counter("processed").Load(),
		adapterTracker.Counter("filled").Load(),
		adapterTracker.ErrorCount(),
		venueRequests.Load())
	t.Log("[s342/RVA-5] PASS — venue rejection does not produce fill (error path proven)")
}

// ---------- RVA-6: Activation Surface Dimensions Correct for Venue Adapter ----------

func TestRealVenueActivation_ActivationSurfaceDimensions(t *testing.T) {
	// Verify the activation surface computation for all three real-venue states:
	// venue_halted, venue_live, venue_degraded.

	// venue + halted + present = venue_halted
	gate := domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s342-rva6-halted",
		UpdatedBy: "s342-test",
		UpdatedAt: time.Now().UTC(),
	}
	surface := domainexec.NewActivationSurface(domainexec.AdapterVenue, gate, domainexec.CredentialPresent)
	if surface.Effective != domainexec.ModeVenueHalted {
		t.Fatalf("[RVA-6/halted] expected venue_halted, got %s", surface.Effective)
	}
	if surface.IsLive() {
		t.Fatal("[RVA-6/halted] halted surface must not be live")
	}
	if !surface.CanReachVenue() {
		t.Fatal("[RVA-6/halted] venue adapter must report CanReachVenue=true")
	}

	// venue + active + present = venue_live
	gate.Status = domainexec.GateActive
	gate.Reason = "s342-rva6-live"
	surface = domainexec.NewActivationSurface(domainexec.AdapterVenue, gate, domainexec.CredentialPresent)
	if surface.Effective != domainexec.ModeVenueLive {
		t.Fatalf("[RVA-6/live] expected venue_live, got %s", surface.Effective)
	}
	if !surface.IsLive() {
		t.Fatal("[RVA-6/live] venue_live must be live")
	}

	// venue + active + absent = venue_degraded
	surface = domainexec.NewActivationSurface(domainexec.AdapterVenue, gate, domainexec.CredentialAbsent)
	if surface.Effective != domainexec.ModeVenueDegraded {
		t.Fatalf("[RVA-6/degraded] expected venue_degraded, got %s", surface.Effective)
	}
	if surface.IsLive() {
		t.Fatal("[RVA-6/degraded] degraded must not be live")
	}

	t.Log("[s342/RVA-6] PASS — activation surface dimensions correct for all venue states")
}
