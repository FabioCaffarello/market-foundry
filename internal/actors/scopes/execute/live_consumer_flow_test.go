//go:build integration

package execute_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	executeactor "internal/actors/scopes/execute"
	natsexecution "internal/adapters/nats/natsexecution"
	"internal/adapters/nats/natskit"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
	natsclient "github.com/nats-io/nats.go"
)

// live_consumer_flow_test.go — S333: NATS Consumer to Actor Live Flow Proof.
//
// This test proves the REAL path from NATS JetStream durable consumer through
// the Hollywood actor system to the VenueAdapterActor, demonstrating that:
//
//   - The ExecuteSupervisor spawns a real JetStream durable consumer
//   - Published PaperOrderSubmittedEvent events are consumed by the durable consumer
//   - The consumer delivers events to VenueAdapterActor via intentReceivedMessage
//   - VenueAdapterActor.onIntent() executes the full safety gate + venue submit pipeline
//   - Fill events are published to EXECUTION_FILL_EVENTS stream
//   - Correlation/causation IDs are preserved end-to-end
//   - Health tracker counters reflect real delivery metrics
//
// Unlike S276 (multi_binary_integration_test.go) which simulates the execute binary
// with core NATS subscriptions and manual onIntent replication, this test exercises
// the REAL actor-based message path through Hollywood.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

// s333Seq ensures unique dedup keys across test runs.
var s333Seq atomic.Int64

// supervisorStartupDelay is the time needed for the JetStream durable consumer
// to fully activate after CreateOrUpdateConsumer + Consume. JetStream push
// consumers have an inherent activation race; this delay is generous to avoid
// flaky failures in CI/test environments.
const supervisorStartupDelay = 1 * time.Second

func s333NatsURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:4222"
	}
	conn, err := net.DialTimeout("tcp", "localhost:4222", 2*time.Second)
	if err != nil {
		t.Skipf("NATS not reachable: %v", err)
	}
	conn.Close()
	return url
}

func s333BuildEvent(t *testing.T, ts time.Time, corrID string) domainexec.PaperOrderSubmittedEvent {
	t.Helper()
	seq := s333Seq.Add(1)
	// Space by full seconds to ensure unique DeduplicationKey (uses .Unix()).
	ts = ts.Add(time.Duration(seq) * time.Second)

	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerpIntegration(t), 60)
	intent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}
	intent.CorrelationID = corrID
	intent.CausationID = "cause-s333-live-flow"

	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("fill simulation should succeed")
	}

	return domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(corrID).
			WithCausationID(intent.CausationID),
		ExecutionIntent: intent,
	}
}

// s333FillSubscriber uses a core NATS subscription on the fill subject space.
// Core NATS subscriptions are immediately active after Flush(), avoiding the
// JetStream ephemeral consumer creation race.
type s333FillSubscriber struct {
	nc       *natsclient.Conn
	sub      *natsclient.Subscription
	registry natsexecution.Registry
	fills    chan domainexec.VenueOrderFilledEvent
}

func newS333FillSubscriber(t *testing.T, url string) *s333FillSubscriber {
	t.Helper()
	nc, err := natsclient.Connect(url)
	if err != nil {
		t.Fatalf("fill subscriber connect: %v", err)
	}

	registry := natsexecution.DefaultRegistry()
	fs := &s333FillSubscriber{
		nc:       nc,
		registry: registry,
		fills:    make(chan domainexec.VenueOrderFilledEvent, 50),
	}

	sub, err := nc.Subscribe("execution.fill.>", func(msg *natsclient.Msg) {
		spec := registry.VenueMarketOrderFilled
		env, prob := natskit.DecodeEvent[domainexec.VenueOrderFilledEvent](spec, msg.Data)
		if prob != nil {
			return
		}
		fs.fills <- env.Payload
	})
	if err != nil {
		nc.Close()
		t.Fatalf("fill subscriber subscribe: %v", err)
	}

	if err := nc.Flush(); err != nil {
		sub.Unsubscribe()
		nc.Close()
		t.Fatalf("fill subscriber flush: %v", err)
	}

	fs.sub = sub
	return fs
}

func (fs *s333FillSubscriber) waitForFill(corrID string, timeout time.Duration) *domainexec.VenueOrderFilledEvent {
	deadline := time.After(timeout)
	for {
		select {
		case fill := <-fs.fills:
			if fill.Metadata.CorrelationID == corrID {
				return &fill
			}
		case <-deadline:
			return nil
		}
	}
}

func (fs *s333FillSubscriber) close() {
	if fs.sub != nil {
		fs.sub.Unsubscribe()
	}
	if fs.nc != nil {
		fs.nc.Close()
	}
}

// s333AppConfig builds a test-safe AppConfig.
func s333AppConfig(url string) settings.AppConfig {
	return settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "paper_simulator",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
}

// s333SetGate sets the execution control gate to the desired state.
func s333SetGate(t *testing.T, url string, status domainexec.GateStatus, reason string) *natsexecution.ControlKVStore {
	t.Helper()
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("control store: %v", err)
	}
	if prob := store.Put(context.Background(), domainexec.ControlGate{
		Status:    status,
		Reason:    reason,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s333-test",
	}); prob != nil {
		t.Fatalf("[s333] set gate %s: %s", status, prob.Message)
	}
	// Confirm the write is server-visible through the actor's read path
	// before the supervisor spawns / events publish (G9 hardening).
	waitGateObserved(t, store, status == domainexec.GateHalted, 5*time.Second)
	return store
}

// s333SpawnSupervisor creates a Hollywood engine, spawns the ExecuteSupervisor, and
// registers cleanup via t.Cleanup to ensure the durable consumer is released even on failure.
func s333SpawnSupervisor(t *testing.T, cfg settings.AppConfig, venue appexec.PaperVenueAdapter, trackers map[string]*healthz.Tracker) *actor.Engine {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(cfg, &venue, nil, trackers),
		fmt.Sprintf("s333-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond) // allow NATS consumer to close
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

// ---------- LF-1: Live Consumer → Actor Flow — Real ExecuteSupervisor ----------

func TestLiveConsumerFlow_RealSupervisorDeliversToActor(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s333-venue-adapter")
	consumerTracker := healthz.NewTracker("s333-venue-consumer")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": consumerTracker,
	}

	controlStore := s333SetGate(t, url, domainexec.GateActive, "s333-lf1")
	defer controlStore.Close()

	// Fill subscriber FIRST (core NATS, immediately active).
	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	// Start the real ExecuteSupervisor (cleanup registered via t.Cleanup).
	venue := *appexec.NewPaperVenueAdapter(0)
	s333SpawnSupervisor(t, s333AppConfig(url), venue, trackers)

	// Separate publisher (simulating derive binary).
	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer publisher.Close()

	// Publish a PaperOrderSubmittedEvent to EXECUTION_EVENTS stream.
	corrID := fmt.Sprintf("s333-lf1-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish execution event: %s", prob.Message)
	}
	t.Logf("[publish] PaperOrderSubmittedEvent published with correlation_id=%s", corrID)

	// Wait for fill event via core NATS subscription.
	fill := fillSub.waitForFill(corrID, 10*time.Second)
	if fill == nil {
		t.Fatal("[fill] fill event not received on EXECUTION_FILL_EVENTS stream — live flow NOT proven")
	}

	// === EVIDENCE: Fill event received on NATS stream ===
	t.Logf("[fill] VenueOrderFilledEvent received: venue_order_id=%s status=%s",
		fill.VenueOrderID, fill.ExecutionIntent.Status)

	// === VALIDATE: Correlation ID preserved end-to-end ===
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("[correlation] want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}
	t.Logf("[correlation] preserved: %s", fill.Metadata.CorrelationID)

	// === VALIDATE: Causation ID links fill to source event ===
	if fill.Metadata.CausationID != event.Metadata.ID {
		t.Fatalf("[causation] fill.CausationID should equal source event Metadata.ID: want %q, got %q",
			event.Metadata.ID, fill.Metadata.CausationID)
	}
	t.Logf("[causation] fill.CausationID=%s links to source event %s", fill.Metadata.CausationID, event.Metadata.ID)

	// === VALIDATE: Fill event domain fields ===
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[status] want filled, got %q", fill.ExecutionIntent.Status)
	}
	if fill.VenueOrderID == "" {
		t.Fatal("[venue_order_id] empty — paper venue adapter did not execute")
	}
	if fill.ExecutionIntent.Source != "binancef" {
		t.Fatalf("[source] want binancef, got %q", fill.ExecutionIntent.Source)
	}
	if fill.ExecutionIntent.VenueSymbol() != "btcusdt" {
		t.Fatalf("[symbol] want btcusdt, got %q", fill.ExecutionIntent.VenueSymbol())
	}
	if len(fill.ExecutionIntent.Fills) == 0 {
		t.Fatal("[fills] no fill records — venue adapter did not populate fills")
	}
	if !fill.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("[fills] expected simulated=true for paper venue")
	}

	// === VALIDATE: Health tracker metrics (consumer side) ===
	consumerEvents := consumerTracker.EventCount()
	if consumerEvents < 1 {
		t.Fatalf("[consumer-tracker] expected EventCount >= 1, got %d", consumerEvents)
	}
	t.Logf("[consumer-tracker] events=%d", consumerEvents)

	// === VALIDATE: Health tracker metrics (adapter side) ===
	// Counters are set by the actor goroutine AFTER PublishFill returns; the
	// NATS subscriber callback above can unblock the test before the actor
	// reaches the Add(1). Eventually-poll over synchronous reads.
	eventuallyAtLeast(t, adapterTracker.Counter("processed"), 1, 2*time.Second,
		"[adapter-tracker] expected processed >= 1")
	eventuallyAtLeast(t, adapterTracker.Counter("filled"), 1, 2*time.Second,
		"[adapter-tracker] expected filled >= 1")
	t.Logf("[adapter-tracker] processed=%d filled=%d",
		adapterTracker.Counter("processed").Load(), adapterTracker.Counter("filled").Load())

	// === VALIDATE: Fill event Metadata.ID is unique ===
	if fill.Metadata.ID == "" {
		t.Fatal("[metadata] fill event Metadata.ID is empty")
	}
	if fill.Metadata.ID == event.Metadata.ID {
		t.Fatal("[metadata] fill event Metadata.ID must differ from source event ID")
	}
	t.Logf("[metadata] fill event ID=%s (distinct from source %s)", fill.Metadata.ID, event.Metadata.ID)

	t.Log("[s333/LF-1] PASS — live consumer → actor flow proven with real ExecuteSupervisor")
}

// ---------- LF-2: Consumer Restart Preserves Durable State ----------

func TestLiveConsumerFlow_ConsumerRestartPreservesDurableState(t *testing.T) {
	url := s333NatsURL(t)
	cfg := s333AppConfig(url)

	controlStore := s333SetGate(t, url, domainexec.GateActive, "s333-lf2")
	defer controlStore.Close()

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// Fill subscriber active the entire test.
	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	// --- Phase 1: Start supervisor, process one event, then stop ---
	tracker1 := healthz.NewTracker("s333-restart-adapter-1")
	engine1, _ := actor.NewEngine(actor.NewEngineConfig())
	venue1 := *appexec.NewPaperVenueAdapter(0)
	pid1 := engine1.Spawn(executeactor.NewExecuteSupervisor(cfg, &venue1, nil,
		map[string]*healthz.Tracker{"venue-adapter": tracker1, "venue-consumer": healthz.NewTracker("c1")}),
		fmt.Sprintf("s333-restart-1-%d", time.Now().UnixNano()),
	)
	defer func() {
		engine1.Poison(pid1)
		time.Sleep(300 * time.Millisecond)
	}()
	time.Sleep(supervisorStartupDelay)

	corrID1 := fmt.Sprintf("s333-lf2-phase1-%d", time.Now().UnixNano())
	event1 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("phase 1 publish: %s", prob.Message)
	}

	fill1 := fillSub.waitForFill(corrID1, 10*time.Second)
	if fill1 == nil {
		t.Fatal("phase 1: fill not received")
	}
	t.Logf("[phase 1] fill received: %s", fill1.VenueOrderID)

	// Stop the first supervisor (simulates binary restart).
	engine1.Poison(pid1)
	time.Sleep(500 * time.Millisecond)
	t.Log("[phase 1] supervisor stopped — simulating restart")

	// --- Phase 2: Restart supervisor, process another event ---
	tracker2 := healthz.NewTracker("s333-restart-adapter-2")
	engine2, _ := actor.NewEngine(actor.NewEngineConfig())
	venue2 := *appexec.NewPaperVenueAdapter(0)
	pid2 := engine2.Spawn(executeactor.NewExecuteSupervisor(cfg, &venue2, nil,
		map[string]*healthz.Tracker{"venue-adapter": tracker2, "venue-consumer": healthz.NewTracker("c2")}),
		fmt.Sprintf("s333-restart-2-%d", time.Now().UnixNano()),
	)
	defer func() {
		engine2.Poison(pid2)
		time.Sleep(300 * time.Millisecond)
	}()
	time.Sleep(supervisorStartupDelay)

	corrID2 := fmt.Sprintf("s333-lf2-phase2-%d", time.Now().UnixNano())
	event2 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("phase 2 publish: %s", prob.Message)
	}

	fill2 := fillSub.waitForFill(corrID2, 10*time.Second)
	if fill2 == nil {
		t.Fatal("phase 2: fill not received after restart — durable state may be lost")
	}
	t.Logf("[phase 2] fill received after restart: %s", fill2.VenueOrderID)

	eventuallyAtLeast(t, tracker2.Counter("filled"), 1, 2*time.Second,
		"phase 2: expected filled >= 1")

	t.Log("[s333/LF-2] PASS — durable consumer resumes after supervisor restart")
}

// ---------- LF-3: Kill Switch Blocks Real Actor Path ----------

func TestLiveConsumerFlow_KillSwitchBlocksRealActorPath(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s333-halt-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s333-halt-consumer"),
	}

	// Set gate to HALTED.
	controlStore := s333SetGate(t, url, domainexec.GateHalted, "s333-lf3-halt")
	defer controlStore.Close()
	defer func() {
		// Restore active gate for subsequent tests.
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s333-lf3-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s333-test",
		})
	}()

	venue := *appexec.NewPaperVenueAdapter(0)
	s333SpawnSupervisor(t, s333AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s333-lf3-halt-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}
	t.Logf("[publish] event published with correlation_id=%s (gate=halted)", corrID)

	// Wait for the consumer to deliver and actor to process.
	// The adapter counter "processed" proves the full path:
	// NATS durable consumer → actor.Send → actor.Receive → onIntent.
	deadline := time.After(10 * time.Second)
	for {
		processedCount := adapterTracker.Counter("processed").Load()
		if processedCount >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("[adapter] event did not reach actor within timeout (processed=%d)", processedCount)
		case <-time.After(100 * time.Millisecond):
		}
	}

	haltedCount := adapterTracker.Counter("skipped_halt").Load()
	filledCount := adapterTracker.Counter("filled").Load()

	if haltedCount < 1 {
		t.Fatalf("[adapter] expected skipped_halt >= 1, got %d — kill switch not working", haltedCount)
	}
	if filledCount != 0 {
		t.Fatalf("[adapter] expected filled=0 when halted, got %d", filledCount)
	}
	t.Logf("[adapter] processed=%d skipped_halt=%d filled=%d — kill switch effective",
		adapterTracker.Counter("processed").Load(), haltedCount, filledCount)

	t.Log("[s333/LF-3] PASS — kill switch blocks real actor path (consumer delivers, actor gates)")
}

// ---------- LF-4: Multiple Events Processed Sequentially ----------

func TestLiveConsumerFlow_MultipleEventsProcessedSequentially(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s333-multi-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s333-multi-consumer"),
	}

	controlStore := s333SetGate(t, url, domainexec.GateActive, "s333-lf4")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	venue := *appexec.NewPaperVenueAdapter(0)
	s333SpawnSupervisor(t, s333AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	const eventCount = 3
	corrIDs := make([]string, eventCount)
	for i := 0; i < eventCount; i++ {
		corrIDs[i] = fmt.Sprintf("s333-lf4-multi-%d-%d", i, time.Now().UnixNano())
	}

	// Publish events with small gaps to avoid dedup window collisions.
	for i := 0; i < eventCount; i++ {
		event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrIDs[i])
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := publisher.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish event %d: %s", i, prob.Message)
		}
		time.Sleep(50 * time.Millisecond) // ensure distinct dedup keys
	}
	t.Logf("[publish] %d events published", eventCount)

	// Collect fills (order may vary due to async processing).
	received := map[string]*domainexec.VenueOrderFilledEvent{}
	deadline := time.After(20 * time.Second)
	for len(received) < eventCount {
		select {
		case fill := <-fillSub.fills:
			for _, cid := range corrIDs {
				if fill.Metadata.CorrelationID == cid {
					received[cid] = &fill
					t.Logf("[fill] received: venue_order_id=%s correlation_id=%s", fill.VenueOrderID, cid)
				}
			}
		case <-deadline:
			t.Fatalf("[fill] timeout: received %d/%d fills", len(received), eventCount)
		}
	}

	// Verify all fills collected with correct status.
	for i, cid := range corrIDs {
		fill := received[cid]
		if fill.ExecutionIntent.Status != domainexec.StatusFilled {
			t.Fatalf("[fill %d] expected filled, got %q", i, fill.ExecutionIntent.Status)
		}
	}

	// Verify tracker counts (eventually — counter trails NATS fill publish).
	eventuallyAtLeast(t, adapterTracker.Counter("filled"), int64(eventCount), 2*time.Second,
		fmt.Sprintf("[adapter] expected filled >= %d", eventCount))
	t.Logf("[adapter] processed=%d filled=%d", adapterTracker.Counter("processed").Load(), adapterTracker.Counter("filled").Load())

	t.Log("[s333/LF-4] PASS — multiple events processed through real actor pipeline")
}
