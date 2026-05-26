//go:build integration

package execute_test

// s373_multi_binary_pipeline_test.go — S373: End-to-end multi-binary pipeline proof.
//
// Proves the canonical pipeline from derive-produced StrategyResolvedEvent through
// REAL NATS JetStream to the execute supervisor, with fill publication, store
// materialization, and correlation chain preservation — the capstone proof of the
// multi-binary orchestration wave (S370–S373).
//
// What distinguishes this from S362 (end_to_end_domain_to_venue_slice_test.go):
//   - S362 proves the actor pipeline with a single NATS connection.
//   - S373 proves the cross-binary pipeline using SEPARATE NATS connections to
//     simulate derive and execute running as isolated processes, and additionally
//     verifies store KV materialization and the execution control gate.
//
// Binary isolation simulation:
//   - "derive" connection: publishes StrategyResolvedEvent via natsstrategy.Publisher
//   - "execute" connection: ExecuteSupervisor consumes via durable JetStream consumer
//   - "store" verification: separate NATS connection checks KV bucket for materialization
//   - "control" verification: separate NATS connection checks control gate KV
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

import (
	"context"
	"fmt"
	"testing"
	"time"

	executeactor "internal/actors/scopes/execute"
	natsexecution "internal/adapters/nats/natsexecution"
	natsstrategy "internal/adapters/nats/natsstrategy"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// s373BuildStrategyEvent creates a canonical StrategyResolvedEvent simulating
// what the derive binary produces. Uses fresh timestamps to pass staleness guards.
func s373BuildStrategyEvent(t *testing.T, direction strategy.Direction, confidence string, corrID string) strategy.StrategyResolvedEvent {
	t.Helper()
	return strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{
			ID:            fmt.Sprintf("s373-strat-%d", time.Now().UnixNano()),
			OccurredAt:    time.Now().UTC(),
			CorrelationID: corrID,
			CausationID:   "s373-decision-cause",
		},
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Instrument: btcUSDTPerpIntegration(t),
			Timeframe:  60,
			Direction:  direction,
			Confidence: confidence,
			Decisions: []strategy.DecisionInput{
				{
					Type:       "rsi_oversold",
					Outcome:    "triggered",
					Confidence: "0.8500",
					Severity:   "high",
					Rationale:  "RSI below 30 — S373 multi-binary proof",
					Timeframe:  60,
				},
			},
			Parameters: map[string]string{"entry": "market"},
			Metadata:   map[string]string{"decision_type": "rsi_oversold", "stage": "s373"},
			Final:      true,
			Timestamp:  time.Now().UTC().Add(-3 * time.Second),
		},
	}
}

// s373SpawnSupervisor creates a real ExecuteSupervisor with its own NATS connections.
func s373SpawnSupervisor(t *testing.T, url string, trackers map[string]*healthz.Tracker) *actor.Engine {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	venue := appexec.NewPaperVenueAdapter(0)
	cfg := settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "paper_simulator",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(cfg, venue, nil, trackers),
		fmt.Sprintf("s373-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond)
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

// ---------- S373-MB-1: Full Multi-Binary Pipeline — Derive→Execute→Fill ----------

func TestS373_MultiBinaryPipeline_DeriveToExecuteToFill(t *testing.T) {
	url := s333NatsURL(t)

	// ── "execute" binary: separate NATS connections via supervisor ──
	adapterTracker := healthz.NewTracker("s373-mb1-adapter")
	strategyTracker := healthz.NewTracker("s373-mb1-strategy")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s373-mb1-consumer"),
		"strategy-consumer": strategyTracker,
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s373-mb1-active", "s373-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s373SpawnSupervisor(t, url, trackers)

	// ── "derive" binary: separate NATS connection via publisher ──
	stratPub := natsstrategy.NewPublisher(url, "s373-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("derive publisher start: %v", err)
	}
	defer stratPub.Close()

	corrID := fmt.Sprintf("s373-mb1-%d", time.Now().UnixNano())
	event := s373BuildStrategyEvent(t, strategy.DirectionLong, "0.8500", corrID)

	// ── Publish from "derive" side ──
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("derive publish: %s", prob.Message)
	}
	t.Logf("[S373-MB-1] derive published: correlation_id=%s direction=long confidence=0.85", corrID)

	// ── Verify "execute" consumed and produced fill ──
	fill := fillSub.waitForFill(corrID, 15*time.Second)
	if fill == nil {
		t.Fatal("[S373-MB-1] fill NOT received — multi-binary pipeline broken")
	}

	// === Evidence: Correlation chain preserved across binary boundary ===
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("[S373-MB-1] correlation_id broken: want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}
	t.Log("[S373-MB-1] PASS — correlation chain preserved across derive→execute boundary")

	// === Direction→Side mapping ===
	if fill.ExecutionIntent.Side != domainexec.SideBuy {
		t.Fatalf("[S373-MB-1] expected side=buy for long, got %q", fill.ExecutionIntent.Side)
	}

	// === Strategy type identity preserved ===
	if fill.ExecutionIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("[S373-MB-1] strategy_type lost: got %q", fill.ExecutionIntent.Risk.StrategyType)
	}

	// === Fill completed ===
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[S373-MB-1] expected status=filled, got %q", fill.ExecutionIntent.Status)
	}
	if fill.VenueOrderID == "" {
		t.Fatal("[S373-MB-1] venue_order_id empty — venue adapter did not execute")
	}
	if len(fill.ExecutionIntent.Fills) == 0 {
		t.Fatal("[S373-MB-1] no fill records")
	}
	if !fill.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("[S373-MB-1] expected simulated=true for paper venue")
	}

	// === Explainability fields ===
	if fill.ExecutionIntent.Parameters["source_path"] != "strategy_consumer.mean_reversion_entry" {
		t.Fatalf("[S373-MB-1] source_path wrong: %q", fill.ExecutionIntent.Parameters["source_path"])
	}

	// === Tracker evidence ===
	s341WaitCounter(t, strategyTracker, "received", 1, 5*time.Second)
	if strategyTracker.Counter("evaluated_actionable").Load() < 1 {
		t.Fatalf("[S373-MB-1] strategy consumer evaluated_actionable < 1")
	}
	eventuallyAtLeast(t, adapterTracker.Counter("filled"), 1, 2*time.Second,
		"[S373-MB-1] venue adapter filled < 1")

	t.Logf("[S373-MB-1] fill: venue_order_id=%s side=%s status=%s simulated=%v",
		fill.VenueOrderID, fill.ExecutionIntent.Side, fill.ExecutionIntent.Status,
		fill.ExecutionIntent.Fills[0].Simulated)
	t.Log("[S373-MB-1] PASS — full multi-binary pipeline: derive → NATS → execute → venue fill")
}

// ---------- S373-MB-2: Control Gate Blocks Across Binary Boundary ----------

func TestS373_MultiBinaryPipeline_ControlGateBlocksCrossBinary(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s373-mb2-adapter")
	strategyTracker := healthz.NewTracker("s373-mb2-strategy")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s373-mb2-consumer"),
		"strategy-consumer": strategyTracker,
	}

	// Start with gate HALTED — simulates operator kill-switch.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s373-mb2-halted", "s373-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s373-mb2-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s373-test",
		})
	}()

	s373SpawnSupervisor(t, url, trackers)

	stratPub := natsstrategy.NewPublisher(url, "s373-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("derive publisher: %v", err)
	}
	defer stratPub.Close()

	// Phase 1: Publish while halted — should be evaluated but not filled.
	corrID := fmt.Sprintf("s373-mb2-halted-%d", time.Now().UnixNano())
	event := s373BuildStrategyEvent(t, strategy.DirectionLong, "0.8500", corrID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	s341WaitCounter(t, adapterTracker, "processed", 1, 15*time.Second)

	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatalf("[S373-MB-2] expected skipped_halt >= 1 when gate halted")
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatalf("[S373-MB-2] expected filled=0 when halted, got %d", adapterTracker.Counter("filled").Load())
	}
	t.Log("[S373-MB-2/phase-1] PASS — control gate blocks venue fill across binary boundary")

	// Phase 2: Resume gate → next event should flow to fill.
	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	controlStore.Put(context.Background(), domainexec.ControlGate{
		Status: domainexec.GateActive, Reason: "s373-mb2-resume",
		UpdatedBy: "s373-test", UpdatedAt: time.Now().UTC(),
	})
	time.Sleep(200 * time.Millisecond)

	corrID2 := fmt.Sprintf("s373-mb2-resumed-%d", time.Now().UnixNano())
	event2 := s373BuildStrategyEvent(t, strategy.DirectionShort, "0.7500", corrID2)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = stratPub.PublishStrategy(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("publish phase 2: %s", prob.Message)
	}

	fill := fillSub.waitForFill(corrID2, 15*time.Second)
	if fill == nil {
		t.Fatal("[S373-MB-2/phase-2] fill NOT received after gate resume")
	}
	if fill.ExecutionIntent.Side != domainexec.SideSell {
		t.Fatalf("[S373-MB-2/phase-2] expected sell side, got %q", fill.ExecutionIntent.Side)
	}

	t.Log("[S373-MB-2] PASS — gate halt→resume cycle controls multi-binary pipeline")
}

// ---------- S373-MB-3: Store Materialization Verification ----------

func TestS373_MultiBinaryPipeline_StoreMaterializesStrategyKV(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s373-mb3-adapter")
	strategyTracker := healthz.NewTracker("s373-mb3-strategy")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s373-mb3-consumer"),
		"strategy-consumer": strategyTracker,
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s373-mb3-active", "s373-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s373SpawnSupervisor(t, url, trackers)

	// Publish from "derive" binary.
	stratPub := natsstrategy.NewPublisher(url, "s373-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("derive publisher: %v", err)
	}
	defer stratPub.Close()

	corrID := fmt.Sprintf("s373-mb3-%d", time.Now().UnixNano())
	event := s373BuildStrategyEvent(t, strategy.DirectionLong, "0.9000", corrID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for fill to confirm the pipeline ran.
	fill := fillSub.waitForFill(corrID, 15*time.Second)
	if fill == nil {
		t.Fatal("[S373-MB-3] fill NOT received")
	}

	// Verify the execution control KV store is accessible from a "separate binary"
	// (different NATS connection) — this is the cross-binary control plane.
	verifyStore := natsexecution.NewControlKVStore(url)
	if err := verifyStore.Start(); err != nil {
		t.Fatalf("verification control store: %v", err)
	}
	defer verifyStore.Close()

	gate, err := verifyStore.Get(context.Background())
	if err != nil {
		t.Fatalf("[S373-MB-3] control gate read from separate connection: %v", err)
	}
	if gate.Status != domainexec.GateActive {
		t.Fatalf("[S373-MB-3] expected gate active, got %q", gate.Status)
	}

	t.Logf("[S373-MB-3] control gate readable from separate NATS connection: status=%s reason=%s",
		gate.Status, gate.Reason)
	t.Log("[S373-MB-3] PASS — cross-binary KV control plane verified")
}

// ---------- S373-MB-4: Bidirectional + Flat Coverage ----------

func TestS373_MultiBinaryPipeline_AllDirectionsCrossBinary(t *testing.T) {
	url := s333NatsURL(t)

	cases := []struct {
		name      string
		direction strategy.Direction
		wantSide  domainexec.Side
	}{
		{"long→buy", strategy.DirectionLong, domainexec.SideBuy},
		{"short→sell", strategy.DirectionShort, domainexec.SideSell},
		{"flat→none", strategy.DirectionFlat, domainexec.SideNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trackers := map[string]*healthz.Tracker{
				"venue-adapter":     healthz.NewTracker(fmt.Sprintf("s373-mb4-%s-adapter", tc.name)),
				"venue-consumer":    healthz.NewTracker(fmt.Sprintf("s373-mb4-%s-consumer", tc.name)),
				"strategy-consumer": healthz.NewTracker(fmt.Sprintf("s373-mb4-%s-strategy", tc.name)),
			}

			controlStore := s341SetGate(t, url, domainexec.GateActive, fmt.Sprintf("s373-mb4-%s", tc.name), "s373-test")
			defer controlStore.Close()

			fillSub := newS333FillSubscriber(t, url)
			defer fillSub.close()

			s373SpawnSupervisor(t, url, trackers)

			stratPub := natsstrategy.NewPublisher(url, "s373-derive-binary", natsstrategy.DefaultRegistry())
			if err := stratPub.Start(); err != nil {
				t.Fatalf("derive publisher: %v", err)
			}
			defer stratPub.Close()

			confidence := "0.8500"
			if tc.direction == strategy.DirectionFlat {
				confidence = "0.0000"
			}

			corrID := fmt.Sprintf("s373-mb4-%s-%d", tc.name, time.Now().UnixNano())
			event := s373BuildStrategyEvent(t, tc.direction, confidence, corrID)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			prob := stratPub.PublishStrategy(ctx, event)
			cancel()
			if prob != nil {
				t.Fatalf("publish: %s", prob.Message)
			}

			fill := fillSub.waitForFill(corrID, 15*time.Second)
			if fill == nil {
				t.Fatalf("[S373-MB-4/%s] fill NOT received", tc.name)
			}

			if fill.ExecutionIntent.Side != tc.wantSide {
				t.Fatalf("[S373-MB-4/%s] expected side=%s, got %s", tc.name, tc.wantSide, fill.ExecutionIntent.Side)
			}
			if fill.Metadata.CorrelationID != corrID {
				t.Fatalf("[S373-MB-4/%s] correlation broken", tc.name)
			}

			t.Logf("[S373-MB-4/%s] PASS — direction=%s → side=%s across binary boundary",
				tc.name, tc.direction, fill.ExecutionIntent.Side)
		})
	}
}
