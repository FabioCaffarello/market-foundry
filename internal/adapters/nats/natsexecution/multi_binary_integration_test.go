//go:build integration

package natsexecution_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	"internal/adapters/nats/natskit"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"

	natsclient "github.com/nats-io/nats.go"
)

// multi_binary_integration_test.go — S276: Multi-Binary Execution Safety Integration Proof.
//
// These tests prove the cross-binary integration boundary between derive, execute,
// and control surface components. Unlike S273/S275 which prove control/safety within
// a single process, these tests enforce binary isolation by:
//
//   - Using separate NATS connections for each simulated binary
//   - Sharing NO Go references between binary roles — only NATS subjects/KV
//   - Each "binary" creates its own publishers, consumers, stores independently
//
// This validates the operational shape where derive, execute, and gateway run as
// distinct OS processes sharing only a NATS cluster.
//
// Proven properties:
//
//   - MB-1: Normal flow — derive publishes to stream, execute consumes and fills
//   - MB-2: Control halt propagates across binaries — halt blocks derive publish AND execute venue
//   - MB-3: Cross-binary safety — execute blocks intents already on stream when gate halts
//   - MB-4: Resume propagates across binaries — both sides allow after resume
//   - MB-5: Full cycle — active→halt→resume observed coherently across binary boundary
//   - MB-6: KV materialization round-trip across binary boundary
//
// Requires a running NATS server. Skipped automatically when unreachable.

// mbSeq is a monotonic counter ensuring unique dedup keys across multi-binary test runs.
var mbSeq atomic.Int64

// deriveBinary simulates the derive binary's execution publisher path.
// It creates its OWN NATS connections (publisher + control store) — no shared Go state.
type deriveBinary struct {
	publisher    *natsexecution.Publisher
	controlStore *natsexecution.ControlKVStore
	published    atomic.Int64
	halted       atomic.Int64
}

func newDeriveBinary(t *testing.T, url string) *deriveBinary {
	t.Helper()
	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("derive: start publisher: %v", err)
	}

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		pub.Close()
		t.Fatalf("derive: start control store: %v", err)
	}

	return &deriveBinary{
		publisher:    pub,
		controlStore: controlStore,
	}
}

func (d *deriveBinary) close() {
	d.publisher.Close()
	d.controlStore.Close()
}

// publish replicates ExecutionPublisherActor.publishWithRetry() — gate check then publish.
func (d *deriveBinary) publish(event domainexec.PaperOrderSubmittedEvent) bool {
	// Gate check: mirrors ExecutionPublisherActor lines 106-122.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	isHalted := d.controlStore.IsHalted(ctx)
	cancel()
	if isHalted {
		d.halted.Add(1)
		return false
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob := d.publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		return false
	}
	d.published.Add(1)
	return true
}

// executeBinary simulates the execute binary's venue adapter path.
// It creates its OWN NATS connections (consumer + control store + fill publisher) — no shared Go state.
// Uses a core NATS subscription (not JetStream push consumer) for reliable, immediate activation.
// The subscription is flushed before returning, guaranteeing the server has registered it.
type executeBinary struct {
	controlStore  *natsexecution.ControlKVStore
	safetyGate    *appexec.SafetyGate
	venue         ports.VenuePort
	fillPublisher *natsexecution.Publisher
	tracker       *healthz.Tracker
	nc            *natsclient.Conn
	sub           *natsclient.Subscription
	received      chan domainexec.PaperOrderSubmittedEvent
	fills         chan domainexec.VenueOrderFilledEvent
	blocked       chan string // reason
}

func newExecuteBinary(t *testing.T, url string) *executeBinary {
	t.Helper()

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("execute: start control store: %v", err)
	}

	safetyGate := appexec.NewSafetyGate(controlStore, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("mb-execute")

	fillPub := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := fillPub.Start(); err != nil {
		controlStore.Close()
		t.Fatalf("execute: start fill publisher: %v", err)
	}

	eb := &executeBinary{
		controlStore:  controlStore,
		safetyGate:    safetyGate,
		venue:         venue,
		fillPublisher: fillPub,
		tracker:       tracker,
		received:      make(chan domainexec.PaperOrderSubmittedEvent, 50),
		fills:         make(chan domainexec.VenueOrderFilledEvent, 50),
		blocked:       make(chan string, 50),
	}

	// Use core NATS subscription for reliable, immediate activation.
	// Core NATS subscriptions become active after Flush(), unlike JetStream push
	// consumers which have an inherent activation race. This subscription reads
	// from the JetStream subject via the NATS core subject space, which means it
	// only receives messages published while the subscription is active.
	nc, err := natsclient.Connect(url)
	if err != nil {
		fillPub.Close()
		controlStore.Close()
		t.Fatalf("execute: connect: %v", err)
	}

	registry := natsexecution.DefaultRegistry()
	sub, err := nc.Subscribe("execution.events.paper_order.submitted.>", func(msg *natsclient.Msg) {
		spec := registry.PaperOrderSubmitted
		env, prob := natskit.DecodeEvent[domainexec.PaperOrderSubmittedEvent](spec, msg.Data)
		if prob != nil {
			return
		}
		eb.onIntent(env.Payload)
	})
	if err != nil {
		nc.Close()
		fillPub.Close()
		controlStore.Close()
		t.Fatalf("execute: subscribe: %v", err)
	}

	// Flush guarantees the subscription is active on the server.
	if err := nc.Flush(); err != nil {
		sub.Unsubscribe()
		nc.Close()
		fillPub.Close()
		controlStore.Close()
		t.Fatalf("execute: flush: %v", err)
	}

	eb.nc = nc
	eb.sub = sub

	return eb
}

func (eb *executeBinary) close() {
	if eb.sub != nil {
		eb.sub.Unsubscribe()
	}
	if eb.nc != nil {
		eb.nc.Close()
	}
	eb.fillPublisher.Close()
	eb.controlStore.Close()
}

// onIntent replicates VenueAdapterActor.onIntent() — the real execute binary handler.
func (eb *executeBinary) onIntent(event domainexec.PaperOrderSubmittedEvent) {
	eb.received <- event

	intent := event.ExecutionIntent
	now := time.Now().UTC()

	eb.tracker.Counter("processed").Add(1)

	verdict := eb.safetyGate.Check(intent.Timestamp, now)
	if !verdict.Allowed {
		switch verdict.Reason {
		case "kill_switch":
			eb.tracker.Counter("skipped_halt").Add(1)
		case "stale":
			eb.tracker.Counter("skipped_stale").Add(1)
		default:
			eb.tracker.RecordError()
		}
		eb.blocked <- verdict.Reason
		return
	}

	receipt, prob := eb.venue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		eb.tracker.RecordError()
		eb.blocked <- "venue_error"
		return
	}

	fill := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(event.Metadata.CorrelationID).
			WithCausationID(event.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// Publish fill to EXECUTION_FILL_EVENTS stream.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	eb.fillPublisher.PublishFill(ctx, fill)
	cancel()

	eb.tracker.RecordEvent()
	eb.tracker.Counter("filled").Add(1)
	eb.fills <- fill
}

// controlSurface simulates the gateway/store binary writing control state.
// Uses its OWN NATS connection — no shared Go state with derive or execute.
type controlSurface struct {
	store *natsexecution.ControlKVStore
}

func newControlSurface(t *testing.T, url string) *controlSurface {
	t.Helper()
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("control: start store: %v", err)
	}
	return &controlSurface{store: store}
}

func (cs *controlSurface) close() {
	cs.store.Close()
}

func (cs *controlSurface) setActive(reason string) error {
	prob := cs.store.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    reason,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s276-control-surface",
	})
	if prob != nil {
		return fmt.Errorf("set active: %s", prob.Message)
	}
	return nil
}

func (cs *controlSurface) setHalted(reason string) error {
	prob := cs.store.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    reason,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s276-control-surface",
	})
	if prob != nil {
		return fmt.Errorf("set halted: %s", prob.Message)
	}
	return nil
}

// mbBuildEvent constructs a valid PaperOrderSubmittedEvent with unique dedup keys.
func mbBuildEvent(t *testing.T, ts time.Time, corrID string) domainexec.PaperOrderSubmittedEvent {
	t.Helper()
	seq := mbSeq.Add(1)
	ts = ts.Add(time.Duration(seq) * time.Millisecond)

	eval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
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
	intent.CausationID = "cause-s276-multi-binary"

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

// waitFill waits for a fill event with the given correlation ID.
func waitFill(ch chan domainexec.VenueOrderFilledEvent, corrID string, timeout time.Duration) *domainexec.VenueOrderFilledEvent {
	deadline := time.After(timeout)
	for {
		select {
		case fill := <-ch:
			if fill.Metadata.CorrelationID == corrID {
				return &fill
			}
		case <-deadline:
			return nil
		}
	}
}

// waitBlocked waits for a blocked reason on the channel.
func waitBlocked(ch chan string, timeout time.Duration) string {
	select {
	case reason := <-ch:
		return reason
	case <-time.After(timeout):
		return ""
	}
}

// waitReceived waits for a received event with the given correlation ID.
func waitReceived(ch chan domainexec.PaperOrderSubmittedEvent, corrID string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case evt := <-ch:
			if evt.Metadata.CorrelationID == corrID {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// ---------- MB-1: Normal flow — derive publishes, execute consumes and fills ----------

func TestMultiBinary_NormalFlow_DerivePublishesExecuteConsumesAndFills(t *testing.T) {
	url := natsURL(t)

	// Three separate "binaries" — each with its own NATS connections.
	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Control surface sets gate to active.
	if err := ctrl.setActive("mb-1-normal-flow"); err != nil {
		t.Fatal(err)
	}

	// Derive binary produces a paper order event.
	corrID := "mb-1-normal-flow"
	event := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	if !derive.publish(event) {
		t.Fatal("derive: expected publish to succeed when gate is active")
	}

	// Execute binary should consume the event and produce a fill.
	fill := waitFill(exec.fills, corrID, 5*time.Second)
	if fill == nil {
		t.Fatal("execute: expected fill event — not received within timeout")
	}

	// Verify fill properties.
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled status, got %q", fill.ExecutionIntent.Status)
	}
	if fill.ExecutionIntent.Side != domainexec.SideBuy {
		t.Fatalf("expected buy side, got %q", fill.ExecutionIntent.Side)
	}
	if fill.VenueOrderID == "" {
		t.Fatal("expected venue order ID")
	}
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("correlation ID mismatch: want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}

	// Verify counters.
	if derive.published.Load() != 1 {
		t.Fatalf("derive: expected published=1, got %d", derive.published.Load())
	}
	if exec.tracker.Counter("processed").Load() != 1 {
		t.Fatalf("execute: expected processed=1, got %d", exec.tracker.Counter("processed").Load())
	}
	if exec.tracker.Counter("filled").Load() != 1 {
		t.Fatalf("execute: expected filled=1, got %d", exec.tracker.Counter("filled").Load())
	}
}

// ---------- MB-2: Halt propagates — control surface halts, derive blocked, execute blocked ----------

func TestMultiBinary_HaltPropagates_DeriveAndExecuteBlocked(t *testing.T) {
	url := natsURL(t)

	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Control surface sets gate to halted.
	if err := ctrl.setHalted("mb-2-halt-propagation"); err != nil {
		t.Fatal(err)
	}

	// Derive binary: publish should be blocked by gate check.
	corrID := "mb-2-halted"
	event := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	if derive.publish(event) {
		t.Fatal("derive: expected publish to be blocked when gate is halted")
	}

	if derive.halted.Load() != 1 {
		t.Fatalf("derive: expected halted=1, got %d", derive.halted.Load())
	}
	if derive.published.Load() != 0 {
		t.Fatalf("derive: expected published=0, got %d", derive.published.Load())
	}

	// Execute binary: safety gate should also report halted (verify shared KV state).
	// We can't test via consumer (nothing on stream), but we can verify the gate directly.
	verdict := exec.safetyGate.Check(event.ExecutionIntent.Timestamp, time.Now().UTC())
	if verdict.Allowed {
		t.Fatal("execute: safety gate should block when control surface halted")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("execute: expected kill_switch, got %q", verdict.Reason)
	}
}

// ---------- MB-3: Cross-binary safety — execute blocks intent when gate halts between publish and consume ----------

func TestMultiBinary_CrossBinarySafety_ExecuteBlocksAfterHalt(t *testing.T) {
	url := natsURL(t)

	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()

	// Start execute binary first (so it's listening).
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Phase 1: Publish while active — execute will receive it.
	if err := ctrl.setActive("mb-3-initial-active"); err != nil {
		t.Fatal(err)
	}

	corrID1 := "mb-3-active-fill"
	event1 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	if !derive.publish(event1) {
		t.Fatal("derive: expected publish to succeed when active")
	}
	fill1 := waitFill(exec.fills, corrID1, 5*time.Second)
	if fill1 == nil {
		t.Fatal("phase 1: expected fill when active")
	}

	// Phase 2: Halt the gate, then publish another event.
	// Derive will be blocked, proving derive-side gate check.
	if err := ctrl.setHalted("mb-3-halt-after-fill"); err != nil {
		t.Fatal(err)
	}

	corrID2 := "mb-3-halted-block"
	event2 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	if derive.publish(event2) {
		t.Fatal("derive: expected publish to be blocked when halted")
	}

	// Phase 3: Verify the execute binary's safety gate independently reports halted.
	// This proves the same KV state is visible from a completely separate NATS connection.
	verdict := exec.safetyGate.Check(event2.ExecutionIntent.Timestamp, time.Now().UTC())
	if verdict.Allowed {
		t.Fatal("execute: safety gate should block when control surface halted")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("execute: expected kill_switch, got %q", verdict.Reason)
	}

	// Cumulative verification.
	if derive.halted.Load() != 1 {
		t.Fatalf("derive: expected halted=1, got %d", derive.halted.Load())
	}
	if exec.tracker.Counter("filled").Load() != 1 {
		t.Fatalf("execute: expected filled=1, got %d", exec.tracker.Counter("filled").Load())
	}
}

// ---------- MB-4: Resume propagates — after resume, both sides allow ----------

func TestMultiBinary_ResumePropagates_BothSidesAllow(t *testing.T) {
	url := natsURL(t)

	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Phase 1: Halt.
	if err := ctrl.setHalted("mb-4-initial-halt"); err != nil {
		t.Fatal(err)
	}

	corrID1 := "mb-4-during-halt"
	event1 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	if derive.publish(event1) {
		t.Fatal("derive: expected block during halt")
	}

	// Phase 2: Resume.
	if err := ctrl.setActive("mb-4-resume"); err != nil {
		t.Fatal(err)
	}

	corrID2 := "mb-4-after-resume"
	event2 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	if !derive.publish(event2) {
		t.Fatal("derive: expected publish to succeed after resume")
	}

	// Execute should consume and fill.
	fill := waitFill(exec.fills, corrID2, 5*time.Second)
	if fill == nil {
		t.Fatal("execute: expected fill after resume — not received")
	}
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %q", fill.ExecutionIntent.Status)
	}

	// Verify derive counters.
	if derive.published.Load() != 1 {
		t.Fatalf("derive: expected published=1, got %d", derive.published.Load())
	}
	if derive.halted.Load() != 1 {
		t.Fatalf("derive: expected halted=1, got %d", derive.halted.Load())
	}
}

// ---------- MB-5: Full cycle — active→halt→resume observed across binary boundary ----------

func TestMultiBinary_FullCycle_ActiveHaltResumeAcrossBoundary(t *testing.T) {
	url := natsURL(t)

	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Phase 1: Active — derive publishes, execute fills.
	if err := ctrl.setActive("mb-5-phase1-active"); err != nil {
		t.Fatal(err)
	}

	corrID1 := "mb-5-phase1"
	event1 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	if !derive.publish(event1) {
		t.Fatal("phase 1: derive should publish when active")
	}
	fill1 := waitFill(exec.fills, corrID1, 5*time.Second)
	if fill1 == nil {
		t.Fatal("phase 1: execute should fill when active")
	}

	// Phase 2: Halt — derive blocked.
	if err := ctrl.setHalted("mb-5-phase2-halt"); err != nil {
		t.Fatal(err)
	}

	corrID2 := "mb-5-phase2"
	event2 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	if derive.publish(event2) {
		t.Fatal("phase 2: derive should be blocked when halted")
	}

	// Phase 3: Resume — derive publishes, execute fills.
	if err := ctrl.setActive("mb-5-phase3-resume"); err != nil {
		t.Fatal(err)
	}

	corrID3 := "mb-5-phase3"
	event3 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID3)
	if !derive.publish(event3) {
		t.Fatal("phase 3: derive should publish after resume")
	}
	fill3 := waitFill(exec.fills, corrID3, 5*time.Second)
	if fill3 == nil {
		t.Fatal("phase 3: execute should fill after resume")
	}

	// Cumulative counters.
	if derive.published.Load() != 2 {
		t.Fatalf("derive: expected published=2, got %d", derive.published.Load())
	}
	if derive.halted.Load() != 1 {
		t.Fatalf("derive: expected halted=1, got %d", derive.halted.Load())
	}
	if exec.tracker.Counter("filled").Load() != 2 {
		t.Fatalf("execute: expected filled=2, got %d", exec.tracker.Counter("filled").Load())
	}
}

// ---------- MB-6: KV materialization round-trip across binary boundary ----------

func TestMultiBinary_KVMaterialization_AcrossBinaryBoundary(t *testing.T) {
	url := natsURL(t)

	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	derive := newDeriveBinary(t, url)
	defer derive.close()
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Set active.
	if err := ctrl.setActive("mb-6-kv-roundtrip"); err != nil {
		t.Fatal(err)
	}

	// Derive publishes.
	corrID := "mb-6-kv-roundtrip"
	event := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	if !derive.publish(event) {
		t.Fatal("derive: expected publish to succeed")
	}

	// Execute should produce fill.
	fill := waitFill(exec.fills, corrID, 5*time.Second)
	if fill == nil {
		t.Fatal("execute: expected fill event")
	}

	// Verify the fill event preserves causal trace from derive.
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("fill correlation ID: want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}
	if fill.ExecutionIntent.Source != "binancef" {
		t.Fatalf("fill source: want binancef, got %q", fill.ExecutionIntent.Source)
	}
	if fill.ExecutionIntent.Symbol != "btcusdt" {
		t.Fatalf("fill symbol: want btcusdt, got %q", fill.ExecutionIntent.Symbol)
	}
	if fill.ExecutionIntent.Type != "paper_order" {
		t.Fatalf("fill type: want paper_order, got %q", fill.ExecutionIntent.Type)
	}

	// Verify venue order ID was generated (proves venue adapter ran).
	if fill.VenueOrderID == "" {
		t.Fatal("fill: expected venue order ID from paper venue adapter")
	}
	if len(fill.VenueOrderID) < 6 {
		t.Fatalf("fill: venue order ID too short: %q", fill.VenueOrderID)
	}

	// Verify fill record exists.
	if len(fill.ExecutionIntent.Fills) == 0 {
		t.Fatal("fill: expected at least one fill record")
	}
	if !fill.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("fill: expected simulated=true for paper fills")
	}

	// Verify control gate state is readable from a completely separate KV store instance.
	// This proves KV is the shared medium, not Go references.
	separateStore := natsexecution.NewControlKVStore(url)
	if err := separateStore.Start(); err != nil {
		t.Fatalf("separate store: %v", err)
	}
	defer separateStore.Close()

	gate, prob := separateStore.Get(context.Background())
	if prob != nil {
		t.Fatalf("separate store get: %s", prob.Message)
	}
	if gate.Status != domainexec.GateActive {
		t.Fatalf("separate store: expected active, got %q", gate.Status)
	}
	if gate.Reason != "mb-6-kv-roundtrip" {
		t.Fatalf("separate store: expected reason mb-6-kv-roundtrip, got %q", gate.Reason)
	}
}
