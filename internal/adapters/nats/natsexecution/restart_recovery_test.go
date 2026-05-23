//go:build integration

package natsexecution_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	"internal/adapters/nats/natskit"
	appexec "internal/application/execution"
	"internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"

	natsclient "github.com/nats-io/nats.go"
)

// restart_recovery_test.go — S280: Durable Restart and Consumer Recovery Proof.
//
// These tests prove that the paper order execution flow recovers correctly
// after component restarts, using real NATS connections. They validate:
//
//   - RR-1: Durable consumer resumes from last ACK after restart
//   - RR-2: Control gate KV state survives consumer reconnect
//   - RR-3: KV projection data persists across store restart
//   - RR-4: Publisher reconnect delivers to restarted consumer
//   - RR-5: Full cycle — publish → restart consumer → resume → verify no loss
//   - RR-6: Execute binary restart — safety gate re-reads KV correctly
//   - RR-7: Dedup boundary — republished events within window are idempotent
//
// Requires a running NATS server. Skipped automatically when unreachable.

// rrSeq is a monotonic counter ensuring unique dedup keys across restart recovery tests.
var rrSeq atomic.Int64

func rrBuildEvent(t *testing.T, ts time.Time, corrID string) execution.PaperOrderSubmittedEvent {
	t.Helper()
	seq := rrSeq.Add(1)
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
	intent.CausationID = "cause-s280-restart-recovery"

	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("fill simulation should succeed")
	}

	return execution.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(corrID).
			WithCausationID(intent.CausationID),
		ExecutionIntent: intent,
	}
}

// ---------- RR-1: Durable consumer resumes from last ACK after restart ----------

func TestRestartRecovery_DurableConsumer_ResumesFromLastACK(t *testing.T) {
	url := natsURL(t)
	if err := natsexecution.ResetExecutionEventsStreamForTest(url); err != nil {
		t.Fatalf("reset EXECUTION_EVENTS stream: %v", err)
	}
	registry := natsexecution.DefaultRegistry()

	// Use a unique durable name to avoid test interference.
	durableName := fmt.Sprintf("rr1-consumer-%d", time.Now().UnixNano())
	spec := natsexecution.WriterPaperOrderExecutionConsumerForTest(durableName)

	// Phase 1: Start publisher and consumer, deliver 3 events, ACK all.
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	received1 := make(chan execution.PaperOrderSubmittedEvent, 10)
	consumer1 := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		received1 <- e
	}, slog.Default())
	if err := consumer1.Start(); err != nil {
		t.Fatalf("start consumer1: %v", err)
	}

	// Publish 3 events.
	baseTS := time.Now().UTC().Add(-30 * time.Second)
	for i := 0; i < 3; i++ {
		corrID := fmt.Sprintf("rr1-pre-restart-%d", i)
		event := rrBuildEvent(t, baseTS, corrID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := pub.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish pre-restart event %d: %s", i, prob.Message)
		}
	}

	// Wait for all 3 to be consumed.
	for i := 0; i < 3; i++ {
		select {
		case <-received1:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for pre-restart event %d", i)
		}
	}

	delivered1, _, _, _ := consumer1.Stats()
	if delivered1 != 3 {
		t.Fatalf("consumer1: expected delivered=3, got %d", delivered1)
	}

	// Phase 2: Stop consumer (simulates restart).
	consumer1.Close()

	// Phase 3: Publish 2 more events while consumer is down.
	for i := 0; i < 2; i++ {
		corrID := fmt.Sprintf("rr1-during-restart-%d", i)
		event := rrBuildEvent(t, baseTS.Add(time.Duration(10+i)*time.Second), corrID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := pub.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish during-restart event %d: %s", i, prob.Message)
		}
	}

	// Phase 4: Start a NEW consumer with the SAME durable name — should resume.
	received2 := make(chan execution.PaperOrderSubmittedEvent, 10)
	consumer2 := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		received2 <- e
	}, slog.Default())
	if err := consumer2.Start(); err != nil {
		t.Fatalf("start consumer2: %v", err)
	}
	defer consumer2.Close()

	// The 2 events published while down should be delivered.
	postRestartCount := 0
	deadline := time.After(10 * time.Second)
	for postRestartCount < 2 {
		select {
		case <-received2:
			postRestartCount++
		case <-deadline:
			t.Fatalf("timeout: expected 2 post-restart events, got %d", postRestartCount)
		}
	}

	// Verify no spurious redelivery of the pre-restart events (they were ACKed).
	select {
	case evt := <-received2:
		// If we get a third event, it should NOT be one of the pre-restart events.
		if evt.Metadata.CorrelationID == "rr1-pre-restart-0" ||
			evt.Metadata.CorrelationID == "rr1-pre-restart-1" ||
			evt.Metadata.CorrelationID == "rr1-pre-restart-2" {
			t.Fatalf("consumer2: received already-ACKed event %q — durable state was lost", evt.Metadata.CorrelationID)
		}
	case <-time.After(500 * time.Millisecond):
		// Good — no extra events.
	}
}

// ---------- RR-2: Control gate KV state survives consumer reconnect ----------

func TestRestartRecovery_ControlGateKV_SurvivesReconnect(t *testing.T) {
	url := natsURL(t)

	// Phase 1: Write halted state with store1.
	store1 := natsexecution.NewControlKVStore(url)
	if err := store1.Start(); err != nil {
		t.Fatalf("start store1: %v", err)
	}

	prob := store1.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "rr-2-pre-restart",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s280-restart-recovery",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	// Phase 2: Close store1 (simulates restart).
	store1.Close()

	// Phase 3: Open a NEW store instance — should read the persisted state.
	store2 := natsexecution.NewControlKVStore(url)
	if err := store2.Start(); err != nil {
		t.Fatalf("start store2: %v", err)
	}
	defer store2.Close()

	gate, prob := store2.Get(context.Background())
	if prob != nil {
		t.Fatalf("get after reconnect: %s", prob.Message)
	}
	if gate.Status != execution.GateHalted {
		t.Fatalf("expected halted, got %q", gate.Status)
	}
	if gate.Reason != "rr-2-pre-restart" {
		t.Fatalf("expected reason rr-2-pre-restart, got %q", gate.Reason)
	}
	if gate.UpdatedBy != "s280-restart-recovery" {
		t.Fatalf("expected updated_by s280-restart-recovery, got %q", gate.UpdatedBy)
	}

	// Phase 4: Update gate to active from store2 — proves write after reconnect works.
	prob = store2.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateActive,
		Reason:    "rr-2-post-restart-resume",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s280-restart-recovery",
	})
	if prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	gate, prob = store2.Get(context.Background())
	if prob != nil {
		t.Fatalf("get after update: %s", prob.Message)
	}
	if gate.Status != execution.GateActive {
		t.Fatalf("expected active after update, got %q", gate.Status)
	}
}

// ---------- RR-3: KV projection data persists across store restart ----------

func TestRestartRecovery_KVProjection_PersistsAcrossRestart(t *testing.T) {
	url := natsURL(t)
	bucket := fmt.Sprintf("TEST_RR3_KV_%d", time.Now().UnixNano())

	// Phase 1: Write execution intent with store1.
	store1 := natsexecution.NewKVStore(url, bucket)
	if err := store1.Start(); err != nil {
		t.Fatalf("start store1: %v", err)
	}

	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	intent := testIntent(ts)
	intent.CorrelationID = "rr-3-persisted"

	result, prob := store1.Put(context.Background(), intent)
	if prob != nil {
		t.Fatalf("put: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten, got %s", result)
	}

	// Phase 2: Close store1 (simulates restart).
	store1.Close()

	// Phase 3: Open a NEW store instance — should read persisted data.
	store2 := natsexecution.NewKVStore(url, bucket)
	if err := store2.Start(); err != nil {
		t.Fatalf("start store2: %v", err)
	}
	defer store2.Close()

	got, prob := store2.Get(context.Background(), "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get after restart: %s", prob.Message)
	}
	if got == nil {
		t.Fatal("expected non-nil intent after restart")
	}
	if got.CorrelationID != "rr-3-persisted" {
		t.Fatalf("expected correlation rr-3-persisted, got %q", got.CorrelationID)
	}
	if !got.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, got.Timestamp)
	}

	// Phase 4: Verify monotonicity guard still works after restart.
	olderTS := ts.Add(-5 * time.Minute)
	olderIntent := testIntent(olderTS)
	result, prob = store2.Put(context.Background(), olderIntent)
	if prob != nil {
		t.Fatalf("put older: %s", prob.Message)
	}
	if result != natskit.PutSkippedStale {
		t.Fatalf("expected PutSkippedStale after restart, got %s", result)
	}
}

// ---------- RR-4: Publisher reconnect delivers to restarted consumer ----------

func TestRestartRecovery_PublisherReconnect_DeliversToRestartedConsumer(t *testing.T) {
	url := natsURL(t)
	if err := natsexecution.ResetExecutionEventsStreamForTest(url); err != nil {
		t.Fatalf("reset EXECUTION_EVENTS stream: %v", err)
	}
	registry := natsexecution.DefaultRegistry()

	durableName := fmt.Sprintf("rr4-consumer-%d", time.Now().UnixNano())
	spec := natsexecution.WriterPaperOrderExecutionConsumerForTest(durableName)

	// Phase 1: Start publisher1 and consumer, deliver one event.
	pub1 := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub1.Start(); err != nil {
		t.Fatalf("start pub1: %v", err)
	}

	received := make(chan execution.PaperOrderSubmittedEvent, 10)
	consumer := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		received <- e
	}, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	event1 := rrBuildEvent(t, time.Now().UTC().Add(-20*time.Second), "rr4-from-pub1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := pub1.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("publish from pub1: %s", prob.Message)
	}

	select {
	case evt := <-received:
		if evt.Metadata.CorrelationID != "rr4-from-pub1" {
			t.Fatalf("expected rr4-from-pub1, got %q", evt.Metadata.CorrelationID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event from pub1")
	}

	// Phase 2: Close publisher1 (simulates publisher restart).
	pub1.Close()

	// Phase 3: Start publisher2 — publish to same stream.
	pub2 := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub2.Start(); err != nil {
		t.Fatalf("start pub2: %v", err)
	}
	defer pub2.Close()

	event2 := rrBuildEvent(t, time.Now().UTC().Add(-10*time.Second), "rr4-from-pub2")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = pub2.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("publish from pub2: %s", prob.Message)
	}

	select {
	case evt := <-received:
		if evt.Metadata.CorrelationID != "rr4-from-pub2" {
			t.Fatalf("expected rr4-from-pub2, got %q", evt.Metadata.CorrelationID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event from pub2")
	}
}

// ---------- RR-5: Full cycle — publish → restart consumer → resume → no loss ----------

func TestRestartRecovery_FullCycle_PublishRestartResumeNoLoss(t *testing.T) {
	url := natsURL(t)
	if err := natsexecution.ResetExecutionEventsStreamForTest(url); err != nil {
		t.Fatalf("reset EXECUTION_EVENTS stream: %v", err)
	}
	registry := natsexecution.DefaultRegistry()

	durableName := fmt.Sprintf("rr5-consumer-%d", time.Now().UnixNano())
	spec := natsexecution.WriterPaperOrderExecutionConsumerForTest(durableName)

	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	// Phase 1: Consumer A processes events 0-4.
	receivedA := make(chan string, 20)
	consumerA := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		receivedA <- e.Metadata.CorrelationID
	}, slog.Default())
	if err := consumerA.Start(); err != nil {
		t.Fatalf("start consumerA: %v", err)
	}

	baseTS := time.Now().UTC().Add(-60 * time.Second)
	for i := 0; i < 5; i++ {
		corrID := fmt.Sprintf("rr5-batch1-%d", i)
		event := rrBuildEvent(t, baseTS.Add(time.Duration(i)*time.Second), corrID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := pub.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish batch1 event %d: %s", i, prob.Message)
		}
	}

	batch1Received := 0
	deadline := time.After(10 * time.Second)
	for batch1Received < 5 {
		select {
		case <-receivedA:
			batch1Received++
		case <-deadline:
			t.Fatalf("timeout: batch1 got %d/5", batch1Received)
		}
	}

	// Phase 2: Stop consumer A.
	consumerA.Close()

	// Phase 3: Publish events 5-9 while consumer is down.
	for i := 5; i < 10; i++ {
		corrID := fmt.Sprintf("rr5-batch2-%d", i)
		event := rrBuildEvent(t, baseTS.Add(time.Duration(i)*time.Second), corrID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := pub.PublishExecution(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish batch2 event %d: %s", i, prob.Message)
		}
	}

	// Phase 4: Start consumer B with same durable name.
	receivedB := make(chan string, 20)
	consumerB := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		receivedB <- e.Metadata.CorrelationID
	}, slog.Default())
	if err := consumerB.Start(); err != nil {
		t.Fatalf("start consumerB: %v", err)
	}
	defer consumerB.Close()

	// Should receive exactly 5 events (batch2).
	batch2IDs := make(map[string]bool)
	deadline = time.After(10 * time.Second)
	for len(batch2IDs) < 5 {
		select {
		case id := <-receivedB:
			batch2IDs[id] = true
		case <-deadline:
			t.Fatalf("timeout: batch2 got %d/5 unique events", len(batch2IDs))
		}
	}

	// Verify all batch2 events arrived.
	for i := 5; i < 10; i++ {
		expected := fmt.Sprintf("rr5-batch2-%d", i)
		if !batch2IDs[expected] {
			t.Errorf("missing batch2 event: %s", expected)
		}
	}

	// Verify no batch1 events were redelivered.
	for id := range batch2IDs {
		for i := 0; i < 5; i++ {
			if id == fmt.Sprintf("rr5-batch1-%d", i) {
				t.Errorf("batch1 event redelivered after restart: %s", id)
			}
		}
	}
}

// ---------- RR-6: Execute binary restart — safety gate re-reads KV correctly ----------

func TestRestartRecovery_ExecuteRestart_SafetyGateReReadsKV(t *testing.T) {
	url := natsURL(t)

	// Phase 1: Control surface sets gate to halted.
	ctrl := natsexecution.NewControlKVStore(url)
	if err := ctrl.Start(); err != nil {
		t.Fatalf("start control: %v", err)
	}
	defer ctrl.Close()

	prob := ctrl.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "rr-6-halt-before-restart",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s280-restart-recovery",
	})
	if prob != nil {
		t.Fatalf("set halted: %s", prob.Message)
	}

	// Phase 2: Simulate execute binary with safety gate — should see halted.
	exec1Store := natsexecution.NewControlKVStore(url)
	if err := exec1Store.Start(); err != nil {
		t.Fatalf("start exec1 store: %v", err)
	}
	safetyGate1 := appexec.NewSafetyGate(exec1Store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	ts := time.Now().UTC().Add(-10 * time.Second)
	verdict := safetyGate1.Check(ts, time.Now().UTC())
	if verdict.Allowed {
		t.Fatal("exec1: expected blocked when halted")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("exec1: expected kill_switch, got %q", verdict.Reason)
	}

	// Phase 3: Close exec1 (simulates execute binary restart).
	exec1Store.Close()

	// Phase 4: Control surface resumes.
	prob = ctrl.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateActive,
		Reason:    "rr-6-resume-after-restart",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s280-restart-recovery",
	})
	if prob != nil {
		t.Fatalf("set active: %s", prob.Message)
	}

	// Phase 5: New execute binary instance — should see active.
	exec2Store := natsexecution.NewControlKVStore(url)
	if err := exec2Store.Start(); err != nil {
		t.Fatalf("start exec2 store: %v", err)
	}
	defer exec2Store.Close()

	safetyGate2 := appexec.NewSafetyGate(exec2Store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	verdict = safetyGate2.Check(ts, time.Now().UTC())
	if !verdict.Allowed {
		t.Fatalf("exec2: expected allowed after resume, got reason %q", verdict.Reason)
	}
}

// ---------- RR-7: Dedup boundary — republished events within window are idempotent ----------

func TestRestartRecovery_DedupBoundary_RepublishedEventsIdempotent(t *testing.T) {
	url := natsURL(t)
	if err := natsexecution.ResetExecutionEventsStreamForTest(url); err != nil {
		t.Fatalf("reset EXECUTION_EVENTS stream: %v", err)
	}
	registry := natsexecution.DefaultRegistry()

	durableName := fmt.Sprintf("rr7-consumer-%d", time.Now().UnixNano())
	spec := natsexecution.WriterPaperOrderExecutionConsumerForTest(durableName)

	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	received := make(chan string, 10)
	consumer := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		received <- e.Metadata.CorrelationID
	}, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	defer consumer.Close()

	// Publish an event.
	ts := time.Now().UTC().Add(-10 * time.Second)
	event := rrBuildEvent(t, ts, "rr7-dedup-test")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := pub.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("first publish: %s", prob.Message)
	}

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first delivery")
	}

	// Republish the SAME event (same dedup key). JetStream should deduplicate.
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = pub.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("second publish: %s", prob.Message)
	}

	// Should NOT receive a second delivery.
	select {
	case id := <-received:
		t.Fatalf("unexpected second delivery of dedup event: %s", id)
	case <-time.After(2 * time.Second):
		// Good — dedup worked.
	}
}

// ---------- RR-8: Multi-binary restart cycle — derive restart, execute survives ----------

func TestRestartRecovery_MultiBinary_DeriveRestartExecuteSurvives(t *testing.T) {
	url := natsURL(t)

	// Control surface.
	ctrl := newControlSurface(t, url)
	defer ctrl.close()
	if err := ctrl.setActive("rr-8-initial"); err != nil {
		t.Fatal(err)
	}

	// Execute binary (stays running throughout).
	exec := newExecuteBinary(t, url)
	defer exec.close()

	// Phase 1: Derive binary 1 publishes — execute receives.
	derive1 := newDeriveBinary(t, url)
	corrID1 := "rr8-from-derive1"
	event1 := mbBuildEvent(t, time.Now().UTC().Add(-20*time.Second), corrID1)
	if !derive1.publish(event1) {
		t.Fatal("derive1: expected publish to succeed")
	}

	fill1 := waitFill(exec.fills, corrID1, 5*time.Second)
	if fill1 == nil {
		t.Fatal("execute: expected fill from derive1")
	}

	// Phase 2: Derive binary 1 crashes.
	derive1.close()

	// Phase 3: Derive binary 2 starts — publishes another event.
	derive2 := newDeriveBinary(t, url)
	defer derive2.close()

	corrID2 := "rr8-from-derive2"
	event2 := mbBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	if !derive2.publish(event2) {
		t.Fatal("derive2: expected publish to succeed")
	}

	fill2 := waitFill(exec.fills, corrID2, 5*time.Second)
	if fill2 == nil {
		t.Fatal("execute: expected fill from derive2")
	}

	// Verify execute received both fills from two different derive instances.
	if exec.tracker.Counter("filled").Load() != 2 {
		t.Fatalf("execute: expected filled=2, got %d", exec.tracker.Counter("filled").Load())
	}
}

// ---------- RR-9: Writer consumer restart with buffered events ----------

func TestRestartRecovery_WriterConsumerDurable_ResumesStreamPosition(t *testing.T) {
	url := natsURL(t)
	if err := natsexecution.ResetExecutionEventsStreamForTest(url); err != nil {
		t.Fatalf("reset EXECUTION_EVENTS stream: %v", err)
	}
	registry := natsexecution.DefaultRegistry()

	// Simulates writer consumer behavior: consume from stream, track position.
	durableName := fmt.Sprintf("rr9-writer-%d", time.Now().UnixNano())
	spec := natsexecution.WriterPaperOrderExecutionConsumerForTest(durableName)

	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	// Phase 1: Consumer processes first batch.
	var countA atomic.Int64
	consumerA := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		countA.Add(1)
	}, slog.Default())
	if err := consumerA.Start(); err != nil {
		t.Fatalf("start consumerA: %v", err)
	}

	baseTS := time.Now().UTC().Add(-60 * time.Second)
	for i := 0; i < 5; i++ {
		event := rrBuildEvent(t, baseTS.Add(time.Duration(i)*time.Second), fmt.Sprintf("rr9-a-%d", i))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pub.PublishExecution(ctx, event)
		cancel()
	}

	// Wait for consumer A to process all.
	deadline := time.After(10 * time.Second)
	for countA.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("consumerA: expected 5, got %d", countA.Load())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Phase 2: Stop consumer A (simulates writer crash — buffer is lost).
	consumerA.Close()

	// Phase 3: Publish more events while writer is down.
	for i := 0; i < 3; i++ {
		event := rrBuildEvent(t, baseTS.Add(time.Duration(10+i)*time.Second), fmt.Sprintf("rr9-b-%d", i))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pub.PublishExecution(ctx, event)
		cancel()
	}

	// Phase 4: Start consumer B (writer recovery).
	var countB atomic.Int64
	consumerB := natsexecution.NewConsumer(url, spec, registry, func(e execution.PaperOrderSubmittedEvent) {
		countB.Add(1)
	}, slog.Default())
	if err := consumerB.Start(); err != nil {
		t.Fatalf("start consumerB: %v", err)
	}
	defer consumerB.Close()

	// Should receive exactly 3 new events (not the 5 already ACKed).
	deadline = time.After(10 * time.Second)
	for countB.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("consumerB: expected 3, got %d", countB.Load())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Verify no over-delivery.
	time.Sleep(500 * time.Millisecond)
	if countB.Load() > 3 {
		t.Fatalf("consumerB: over-delivery detected — got %d instead of 3", countB.Load())
	}
}

// ---------- RR-10: Control gate survives cross-binary restart ----------

func TestRestartRecovery_ControlGate_CrossBinaryRestartCoherent(t *testing.T) {
	url := natsURL(t)

	// Phase 1: Store binary writes gate.
	storeBinary := natsexecution.NewControlKVStore(url)
	if err := storeBinary.Start(); err != nil {
		t.Fatalf("start store binary: %v", err)
	}

	tracker := healthz.NewTracker("rr10-test")
	_ = tracker

	prob := storeBinary.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateHalted,
		Reason:    "rr-10-maintenance",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "rr-10-store-binary",
	})
	if prob != nil {
		t.Fatalf("store put: %s", prob.Message)
	}

	// Phase 2: Derive binary reads gate.
	deriveBinaryStore := natsexecution.NewControlKVStore(url)
	if err := deriveBinaryStore.Start(); err != nil {
		t.Fatalf("start derive store: %v", err)
	}
	if !deriveBinaryStore.IsHalted(context.Background()) {
		t.Fatal("derive: expected halted")
	}

	// Phase 3: Store binary restarts.
	storeBinary.Close()
	storeBinary2 := natsexecution.NewControlKVStore(url)
	if err := storeBinary2.Start(); err != nil {
		t.Fatalf("restart store binary: %v", err)
	}
	defer storeBinary2.Close()

	// Phase 4: Store binary 2 writes active.
	prob = storeBinary2.Put(context.Background(), execution.ControlGate{
		Status:    execution.GateActive,
		Reason:    "rr-10-maintenance-over",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "rr-10-store-binary-2",
	})
	if prob != nil {
		t.Fatalf("store2 put: %s", prob.Message)
	}

	// Phase 5: Derive binary (still running) reads the update.
	if deriveBinaryStore.IsHalted(context.Background()) {
		t.Fatal("derive: expected active after store restart and update")
	}
	deriveBinaryStore.Close()

	// Phase 6: New execute binary connects and reads the correct state.
	executeBinaryStore := natsexecution.NewControlKVStore(url)
	if err := executeBinaryStore.Start(); err != nil {
		t.Fatalf("start execute store: %v", err)
	}
	defer executeBinaryStore.Close()

	gate, prob := executeBinaryStore.Get(context.Background())
	if prob != nil {
		t.Fatalf("execute get: %s", prob.Message)
	}
	if gate.Status != execution.GateActive {
		t.Fatalf("execute: expected active, got %q", gate.Status)
	}
	if gate.UpdatedBy != "rr-10-store-binary-2" {
		t.Fatalf("execute: expected updated_by rr-10-store-binary-2, got %q", gate.UpdatedBy)
	}
}

// ---------- Helpers (reuse core NATS sub pattern from multi_binary tests) ----------

func rrWaitEvent(ch chan execution.PaperOrderSubmittedEvent, corrID string, timeout time.Duration) bool {
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

// rrSubscribeCoreNATS creates a core NATS subscription on the execution subject.
// Returns the subscription and a channel for received events.
func rrSubscribeCoreNATS(t *testing.T, url string) (*natsclient.Conn, *natsclient.Subscription, chan execution.PaperOrderSubmittedEvent) {
	t.Helper()
	nc, err := natsclient.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	registry := natsexecution.DefaultRegistry()
	ch := make(chan execution.PaperOrderSubmittedEvent, 50)
	sub, err := nc.Subscribe("execution.events.paper_order.submitted.>", func(msg *natsclient.Msg) {
		spec := registry.PaperOrderSubmitted
		env, prob := natskit.DecodeEvent[execution.PaperOrderSubmittedEvent](spec, msg.Data)
		if prob != nil {
			return
		}
		ch <- env.Payload
	})
	if err != nil {
		nc.Close()
		t.Fatalf("subscribe: %v", err)
	}
	if err := nc.Flush(); err != nil {
		sub.Unsubscribe()
		nc.Close()
		t.Fatalf("flush: %v", err)
	}

	return nc, sub, ch
}
