//go:build integration

package nats_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	adapternats "internal/adapters/nats"
	domainexec "internal/domain/execution"
	"internal/shared/events"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

// startEmbeddedNATS starts an in-process NATS server with JetStream enabled.
// Returns the server and the client URL. The server is automatically shut down on test cleanup.
func startEmbeddedNATS(t *testing.T) string {
	t.Helper()

	opts := &natsserver.Options{
		Host:           "127.0.0.1",
		Port:           -1, // random port
		NoLog:          true,
		NoSigs:         true,
		JetStream:      true,
		StoreDir:       t.TempDir(),
	}

	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("create embedded NATS server: %v", err)
	}
	srv.Start()

	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("embedded NATS server did not become ready")
	}

	t.Cleanup(func() {
		srv.Shutdown()
		srv.WaitForShutdown()
	})

	return srv.ClientURL()
}

func testRegistry() adapternats.ExecutionRegistry {
	return adapternats.DefaultExecutionRegistry()
}

func testPaperOrderEvent(source, symbol string, timeframe int) domainexec.PaperOrderSubmittedEvent {
	return domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("corr-integ-" + symbol).
			WithCausationID("cause-integ-" + symbol),
		ExecutionIntent: domainexec.ExecutionIntent{
			Type:          "paper_order",
			Source:        source,
			Symbol:        symbol,
			Timeframe:     timeframe,
			Side:          domainexec.SideBuy,
			Quantity:      "0.001",
			Status:        domainexec.StatusFilled,
			CorrelationID: "corr-integ-" + symbol,
			CausationID:   "cause-integ-" + symbol,
			Risk: domainexec.RiskInput{
				Type:        "position_exposure",
				Disposition: "approved",
				Confidence:  "0.85",
				Timeframe:   timeframe,
			},
			FilledQuantity: "0.001",
			Fills: []domainexec.FillRecord{
				{Price: "0", Quantity: "0.001", Fee: "0", Simulated: true, Timestamp: time.Now().UTC()},
			},
			Final:     true,
			Timestamp: time.Now().UTC(),
		},
	}
}

func testFillEvent(source, symbol string, timeframe int, venueOrderID string) domainexec.VenueOrderFilledEvent {
	return domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("corr-fill-" + symbol).
			WithCausationID("cause-fill-" + symbol),
		ExecutionIntent: domainexec.ExecutionIntent{
			Type:           "paper_order",
			Source:         source,
			Symbol:         symbol,
			Timeframe:      timeframe,
			Side:           domainexec.SideBuy,
			Quantity:       "0.001",
			FilledQuantity: "0.001",
			Status:         domainexec.StatusFilled,
			CorrelationID:  "corr-fill-" + symbol,
			CausationID:    "cause-fill-" + symbol,
			Risk: domainexec.RiskInput{
				Type:        "position_exposure",
				Disposition: "approved",
				Confidence:  "0.85",
				Timeframe:   timeframe,
			},
			Fills: []domainexec.FillRecord{
				{Price: "65000.00", Quantity: "0.001", Fee: "0.065", Simulated: false, Timestamp: time.Now().UTC()},
			},
			Final:     true,
			Timestamp: time.Now().UTC(),
		},
		VenueOrderID: venueOrderID,
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 1: Publish execution event → consumer receives it
// ──────────────────────────────────────────────────────────────────

func TestIntegration_PublishExecution_ConsumerReceives(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	// Start publisher.
	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	// Start consumer with capture handler.
	var received domainexec.PaperOrderSubmittedEvent
	var mu sync.Mutex
	done := make(chan struct{})

	handler := func(event domainexec.PaperOrderSubmittedEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = event
		close(done)
	}

	spec := adapternats.ExecuteVenueMarketOrderIntakeConsumer()
	consumer := adapternats.NewExecutionConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish event.
	event := testPaperOrderEvent("binancef", "btcusdt", 60)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := pub.PublishExecution(ctx, event); prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for delivery.
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for consumer delivery")
	}

	mu.Lock()
	defer mu.Unlock()

	if received.ExecutionIntent.Symbol != "btcusdt" {
		t.Fatalf("expected btcusdt, got %s", received.ExecutionIntent.Symbol)
	}
	if received.Metadata.CorrelationID != event.Metadata.CorrelationID {
		t.Fatal("correlation ID not preserved through publish/consume")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 2: Publish fill event → fill consumer receives it
// ──────────────────────────────────────────────────────────────────

func TestIntegration_PublishFill_FillConsumerReceives(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	var received domainexec.VenueOrderFilledEvent
	var mu sync.Mutex
	done := make(chan struct{})

	handler := func(event domainexec.VenueOrderFilledEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = event
		close(done)
	}

	spec := adapternats.StoreVenueMarketOrderFillConsumer()
	consumer := adapternats.NewFillConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start fill consumer: %v", err)
	}
	defer consumer.Close()

	fillEvent := testFillEvent("binancef", "btcusdt", 60, "venue-order-001")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := pub.PublishFill(ctx, fillEvent); prob != nil {
		t.Fatalf("publish fill: %s", prob.Message)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for fill consumer delivery")
	}

	mu.Lock()
	defer mu.Unlock()

	if received.VenueOrderID != "venue-order-001" {
		t.Fatalf("expected venue-order-001, got %s", received.VenueOrderID)
	}
	if received.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("fill should not be simulated")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 3: KV store put/get for execution latest (projection)
// ──────────────────────────────────────────────────────────────────

func TestIntegration_ExecutionKV_PutGet(t *testing.T) {
	url := startEmbeddedNATS(t)

	store := adapternats.NewExecutionKVStore(url, adapternats.ExecutionPaperOrderLatestBucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	intent := domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		FilledQuantity: "0.001",
		Status:    domainexec.StatusFilled,
		Risk: domainexec.RiskInput{
			Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60,
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}

	ctx := context.Background()

	result, prob := store.Put(ctx, intent)
	if prob != nil {
		t.Fatalf("put: %s", prob.Message)
	}
	if result != adapternats.PutWritten {
		t.Fatalf("expected PutWritten, got %v", result)
	}

	got, prob := store.Get(ctx, "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if got == nil {
		t.Fatal("expected non-nil intent")
	}
	if got.Symbol != "btcusdt" {
		t.Fatalf("expected btcusdt, got %s", got.Symbol)
	}
	if got.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", got.Status)
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 4: KV monotonicity guard — stale and duplicate rejection
// ──────────────────────────────────────────────────────────────────

func TestIntegration_ExecutionKV_MonotonicityGuard(t *testing.T) {
	url := startEmbeddedNATS(t)

	store := adapternats.NewExecutionKVStore(url, adapternats.ExecutionPaperOrderLatestBucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	base := domainexec.ExecutionIntent{
		Type: "paper_order", Source: "binancef", Symbol: "ethusdt", Timeframe: 60,
		Side: domainexec.SideBuy, Quantity: "0.01", FilledQuantity: "0.01",
		Status: domainexec.StatusFilled, Final: true,
		Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		Timestamp: now,
	}

	// Write initial.
	result, _ := store.Put(ctx, base)
	if result != adapternats.PutWritten {
		t.Fatalf("first put: expected PutWritten, got %v", result)
	}

	// Attempt to write older timestamp → should be skipped as stale.
	stale := base
	stale.Timestamp = now.Add(-1 * time.Minute)
	result, _ = store.Put(ctx, stale)
	if result != adapternats.PutSkippedStale {
		t.Fatalf("stale put: expected PutSkippedStale, got %v", result)
	}

	// Attempt to write same timestamp → should be skipped as duplicate.
	dup := base
	result, _ = store.Put(ctx, dup)
	if result != adapternats.PutSkippedDuplicate {
		t.Fatalf("dup put: expected PutSkippedDuplicate, got %v", result)
	}

	// Write newer → should succeed.
	newer := base
	newer.Timestamp = now.Add(1 * time.Minute)
	result, _ = store.Put(ctx, newer)
	if result != adapternats.PutWritten {
		t.Fatalf("newer put: expected PutWritten, got %v", result)
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 5: Control gate KV — active/halted lifecycle
// ──────────────────────────────────────────────────────────────────

func TestIntegration_ControlGate_Lifecycle(t *testing.T) {
	url := startEmbeddedNATS(t)

	store := adapternats.NewExecutionControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Default gate should be active.
	gate, prob := store.Get(ctx)
	if prob != nil {
		t.Fatalf("get default: %s", prob.Message)
	}
	if gate.IsHalted() {
		t.Fatal("default gate should be active")
	}

	// Halt the gate.
	halted := domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "integration test halt",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "test",
	}
	if prob := store.Put(ctx, halted); prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	// Verify halted.
	gate, _ = store.Get(ctx)
	if !gate.IsHalted() {
		t.Fatal("gate should be halted")
	}
	if gate.Reason != "integration test halt" {
		t.Fatalf("expected halt reason, got %q", gate.Reason)
	}

	// Re-activate.
	active := domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "re-enabled after test",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "test",
	}
	if prob := store.Put(ctx, active); prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	// IsHalted helper check.
	if store.IsHalted(ctx) {
		t.Fatal("gate should be active again")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 6: Full publish → consume → project to KV pipeline
// ──────────────────────────────────────────────────────────────────

func TestIntegration_PublishConsumeProject_Pipeline(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	// Start publisher.
	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	// Start KV store for projection.
	kvStore := adapternats.NewExecutionKVStore(url, adapternats.ExecutionPaperOrderLatestBucket)
	if err := kvStore.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer kvStore.Close()

	// Simulated projection handler: consume → project to KV.
	var wg sync.WaitGroup
	wg.Add(1)
	handler := func(event domainexec.PaperOrderSubmittedEvent) {
		defer wg.Done()
		intent := event.ExecutionIntent
		if !intent.Final {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, prob := kvStore.Put(ctx, intent); prob != nil {
			t.Errorf("projection put: %s", prob.Message)
		}
	}

	spec := adapternats.StorePaperOrderExecutionConsumer()
	consumer := adapternats.NewExecutionConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish.
	event := testPaperOrderEvent("binancef", "btcusdt", 60)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if prob := pub.PublishExecution(ctx, event); prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for projection to complete.
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for projection")
	}

	// Verify projected KV entry.
	got, prob := kvStore.Get(context.Background(), "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get projected: %s", prob.Message)
	}
	if got == nil {
		t.Fatal("projection did not write to KV")
	}
	if got.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", got.Status)
	}
	if got.CorrelationID != event.Metadata.CorrelationID {
		t.Fatal("correlation ID not preserved through pipeline")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 7: JetStream deduplication — same event published twice
// ──────────────────────────────────────────────────────────────────

func TestIntegration_JetStream_Deduplication(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	var deliveryCount int
	var mu sync.Mutex
	deliveries := make(chan struct{}, 10)

	handler := func(event domainexec.PaperOrderSubmittedEvent) {
		mu.Lock()
		deliveryCount++
		mu.Unlock()
		deliveries <- struct{}{}
	}

	spec := adapternats.StorePaperOrderExecutionConsumer()
	consumer := adapternats.NewExecutionConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish the SAME event twice (same dedup key).
	event := testPaperOrderEvent("binancef", "btcusdt", 60)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := pub.PublishExecution(ctx, event); prob != nil {
		t.Fatalf("first publish: %s", prob.Message)
	}
	if prob := pub.PublishExecution(ctx, event); prob != nil {
		t.Fatalf("second publish: %s", prob.Message)
	}

	// Wait for first delivery.
	select {
	case <-deliveries:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for delivery")
	}

	// Give extra time for any spurious second delivery.
	time.Sleep(2 * time.Second)

	mu.Lock()
	count := deliveryCount
	mu.Unlock()

	if count != 1 {
		t.Fatalf("expected exactly 1 delivery (dedup), got %d", count)
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 8: Multi-symbol isolation — events route to correct keys
// ──────────────────────────────────────────────────────────────────

func TestIntegration_MultiSymbol_Isolation(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	kvStore := adapternats.NewExecutionKVStore(url, adapternats.ExecutionPaperOrderLatestBucket)
	if err := kvStore.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer kvStore.Close()

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	var wg sync.WaitGroup
	wg.Add(len(symbols))

	handler := func(event domainexec.PaperOrderSubmittedEvent) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		kvStore.Put(ctx, event.ExecutionIntent)
	}

	spec := adapternats.StorePaperOrderExecutionConsumer()
	consumer := adapternats.NewExecutionConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish events for each symbol.
	for _, sym := range symbols {
		event := testPaperOrderEvent("binancef", sym, 60)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if prob := pub.PublishExecution(ctx, event); prob != nil {
			cancel()
			t.Fatalf("publish %s: %s", sym, prob.Message)
		}
		cancel()
	}

	// Wait for all projections.
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for multi-symbol projections")
	}

	// Verify each symbol has its own isolated KV entry.
	ctx := context.Background()
	for _, sym := range symbols {
		got, prob := kvStore.Get(ctx, "binancef", sym, 60)
		if prob != nil {
			t.Fatalf("get %s: %s", sym, prob.Message)
		}
		if got == nil {
			t.Fatalf("no KV entry for %s", sym)
		}
		if got.Symbol != sym {
			t.Fatalf("symbol bleed: expected %s, got %s", sym, got.Symbol)
		}
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 9: Fill pipeline — publish fill → consume → project
// ──────────────────────────────────────────────────────────────────

func TestIntegration_FillPipeline_PublishConsumeProject(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	kvStore := adapternats.NewExecutionKVStore(url, adapternats.ExecutionVenueMarketOrderLatestBucket)
	if err := kvStore.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer kvStore.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	fillHandler := func(event domainexec.VenueOrderFilledEvent) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		kvStore.Put(ctx, event.ExecutionIntent)
	}

	spec := adapternats.StoreVenueMarketOrderFillConsumer()
	consumer := adapternats.NewFillConsumer(url, spec, registry, fillHandler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start fill consumer: %v", err)
	}
	defer consumer.Close()

	fillEvent := testFillEvent("binancef", "btcusdt", 60, "venue-123")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := pub.PublishFill(ctx, fillEvent); prob != nil {
		t.Fatalf("publish fill: %s", prob.Message)
	}

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for fill projection")
	}

	got, prob := kvStore.Get(context.Background(), "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get fill projection: %s", prob.Message)
	}
	if got == nil {
		t.Fatal("fill projection not written")
	}
	if got.Fills[0].Price != "65000.00" {
		t.Fatalf("expected price 65000.00, got %s", got.Fills[0].Price)
	}
	if got.Fills[0].Simulated {
		t.Fatal("real fill should not be simulated")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 10: Control gate blocks → re-enables (kill switch flow)
// ──────────────────────────────────────────────────────────────────

func TestIntegration_ControlGate_BlockAndResume(t *testing.T) {
	url := startEmbeddedNATS(t)

	store := adapternats.NewExecutionControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Initially active.
	if store.IsHalted(ctx) {
		t.Fatal("should start active")
	}

	// Halt.
	store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "emergency stop",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "operator",
	})

	if !store.IsHalted(ctx) {
		t.Fatal("should be halted after put")
	}

	// Simulate multiple reads (cache-free verification).
	for i := 0; i < 5; i++ {
		if !store.IsHalted(ctx) {
			t.Fatalf("iteration %d: expected halted", i)
		}
	}

	// Resume.
	store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "all clear",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "operator",
	})

	if store.IsHalted(ctx) {
		t.Fatal("should be active after resume")
	}
}

// ──────────────────────────────────────────────────────────────────
// Scenario 11: Consumer stats tracking
// ──────────────────────────────────────────────────────────────────

func TestIntegration_ConsumerStats_Tracking(t *testing.T) {
	url := startEmbeddedNATS(t)
	registry := testRegistry()

	pub := adapternats.NewExecutionPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	var mu sync.Mutex
	count := 0
	done := make(chan struct{})
	target := 3

	handler := func(event domainexec.PaperOrderSubmittedEvent) {
		mu.Lock()
		count++
		if count >= target {
			close(done)
		}
		mu.Unlock()
	}

	spec := adapternats.StorePaperOrderExecutionConsumer()
	consumer := adapternats.NewExecutionConsumer(url, spec, registry, handler, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish 3 distinct events.
	for i := 0; i < target; i++ {
		event := testPaperOrderEvent("binancef", fmt.Sprintf("sym%d", i), 60)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if prob := pub.PublishExecution(ctx, event); prob != nil {
			cancel()
			t.Fatalf("publish %d: %s", i, prob.Message)
		}
		cancel()
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for all deliveries")
	}

	delivered, redelivered, terminated, nakked := consumer.Stats()
	if delivered != int64(target) {
		t.Fatalf("expected %d delivered, got %d", target, delivered)
	}
	if redelivered != 0 {
		t.Fatalf("expected 0 redelivered, got %d", redelivered)
	}
	if terminated != 0 {
		t.Fatalf("expected 0 terminated, got %d", terminated)
	}
	if nakked != 0 {
		t.Fatalf("expected 0 nakked, got %d", nakked)
	}
}
