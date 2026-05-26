//go:build integration

package execute_test

// s380_live_listen_dry_run_test.go — S380: End-to-end live-listen + dry-run proof.
//
// Proves the canonical pipeline from derive-produced StrategyResolvedEvent through
// REAL NATS JetStream to the execute supervisor, with the DryRunSubmitter active
// (the production default), verifying that:
//
//   - Strategy events are consumed across the binary boundary
//   - DryRunSubmitter intercepts all venue calls (no real venue contact)
//   - Fill events carry "dryrun-" prefix and Simulated=true
//   - Correlation/causation chain is fully preserved
//   - Health counters track dry-run interceptions
//
// This is the capstone proof of the exchange listening + dry-run foundation wave
// (S376–S381). It combines S373's cross-binary pipeline with S379's dry-run
// submitter in a single integrated validation.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	executeactor "internal/actors/scopes/execute"
	natsstrategy "internal/adapters/nats/natsstrategy"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// s380SpawnDryRunSupervisor creates an ExecuteSupervisor with DryRunSubmitter
// as outermost decorator — matching the production wiring in cmd/execute/run.go.
func s380SpawnDryRunSupervisor(t *testing.T, url string, trackers map[string]*healthz.Tracker) *actor.Engine {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	// Build venue pipeline: paper → DryRunSubmitter (outermost).
	// This matches cmd/execute/run.go when venue.dry_run=true (default).
	rawVenue := appexec.NewPaperVenueAdapter(0)
	dryRunSubmitter := appexec.NewDryRunSubmitter(rawVenue).
		WithTracker(trackers["venue-adapter"])

	cfg := settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "paper_simulator",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(cfg, dryRunSubmitter, nil, trackers),
		fmt.Sprintf("s380-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond)
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

func s380BuildStrategyEvent(t *testing.T, direction strategy.Direction, confidence string, corrID string) strategy.StrategyResolvedEvent {
	t.Helper()
	return strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{
			ID:            fmt.Sprintf("s380-strat-%d", time.Now().UnixNano()),
			OccurredAt:    time.Now().UTC(),
			CorrelationID: corrID,
			CausationID:   "s380-decision-cause",
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
					Rationale:  "RSI below 30 — S380 live-listen + dry-run proof",
					Timeframe:  60,
				},
			},
			Parameters: map[string]string{"entry": "market"},
			Metadata:   map[string]string{"decision_type": "rsi_oversold", "stage": "s380"},
			Final:      true,
			Timestamp:  time.Now().UTC().Add(-3 * time.Second),
		},
	}
}

// ---------- S380-DR-1: Full Pipeline with DryRunSubmitter — Derive→Execute→DryRun Fill ----------

func TestS380_LiveListenDryRun_FullPipeline(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s380-dr1-adapter")
	strategyTracker := healthz.NewTracker("s380-dr1-strategy")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s380-dr1-consumer"),
		"strategy-consumer": strategyTracker,
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s380-dr1-active", "s380-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s380SpawnDryRunSupervisor(t, url, trackers)

	// "derive" binary: separate NATS connection.
	stratPub := natsstrategy.NewPublisher(url, "s380-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("derive publisher start: %v", err)
	}
	defer stratPub.Close()

	corrID := fmt.Sprintf("s380-dr1-%d", time.Now().UnixNano())
	event := s380BuildStrategyEvent(t, strategy.DirectionLong, "0.8500", corrID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("derive publish: %s", prob.Message)
	}
	t.Logf("[S380-DR-1] derive published: correlation_id=%s direction=long", corrID)

	fill := fillSub.waitForFill(corrID, 15*time.Second)
	if fill == nil {
		t.Fatal("[S380-DR-1] fill NOT received — dry-run pipeline broken")
	}

	// === Evidence: DryRunSubmitter produced the fill ===
	if !strings.HasPrefix(fill.VenueOrderID, "dryrun-") {
		t.Fatalf("[S380-DR-1] venue_order_id must have dryrun- prefix, got: %s", fill.VenueOrderID)
	}
	t.Logf("[S380-DR-1] venue_order_id=%s (dryrun- prefix confirmed)", fill.VenueOrderID)

	// === Simulated flag on fill records ===
	if len(fill.ExecutionIntent.Fills) == 0 {
		t.Fatal("[S380-DR-1] no fill records")
	}
	if !fill.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("[S380-DR-1] fill must be marked Simulated=true")
	}

	// === Correlation chain preserved ===
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("[S380-DR-1] correlation_id broken: want %q, got %q", corrID, fill.Metadata.CorrelationID)
	}

	// === Status and side ===
	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[S380-DR-1] expected status=filled, got %s", fill.ExecutionIntent.Status)
	}
	if fill.ExecutionIntent.Side != domainexec.SideBuy {
		t.Fatalf("[S380-DR-1] expected side=buy for long, got %s", fill.ExecutionIntent.Side)
	}

	// === Strategy type identity preserved ===
	if fill.ExecutionIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("[S380-DR-1] strategy_type lost: got %q", fill.ExecutionIntent.Risk.StrategyType)
	}

	// === Health counters ===
	s341WaitCounter(t, strategyTracker, "received", 1, 5*time.Second)
	if strategyTracker.Counter("evaluated_actionable").Load() < 1 {
		t.Fatal("[S380-DR-1] strategy consumer evaluated_actionable < 1")
	}
	if adapterTracker.Counter("dryrun_intercepted").Load() < 1 {
		t.Fatal("[S380-DR-1] dryrun_intercepted counter not incremented")
	}
	if adapterTracker.Counter("dryrun_filled").Load() < 1 {
		t.Fatal("[S380-DR-1] dryrun_filled counter not incremented")
	}

	t.Logf("[S380-DR-1] fill: venue_order_id=%s side=%s status=%s simulated=%v dryrun_intercepted=%d",
		fill.VenueOrderID, fill.ExecutionIntent.Side, fill.ExecutionIntent.Status,
		fill.ExecutionIntent.Fills[0].Simulated, adapterTracker.Counter("dryrun_intercepted").Load())
	t.Log("[S380-DR-1] PASS — full pipeline: derive → NATS → execute → DryRunSubmitter → dry-run fill")
}

// ---------- S380-DR-2: DryRun No-Action (Flat Direction) ----------

func TestS380_LiveListenDryRun_FlatDirectionNoAction(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s380-dr2-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s380-dr2-consumer"),
		"strategy-consumer": healthz.NewTracker("s380-dr2-strategy"),
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s380-dr2-active", "s380-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s380SpawnDryRunSupervisor(t, url, trackers)

	stratPub := natsstrategy.NewPublisher(url, "s380-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("derive publisher: %v", err)
	}
	defer stratPub.Close()

	corrID := fmt.Sprintf("s380-dr2-%d", time.Now().UnixNano())
	event := s380BuildStrategyEvent(t, strategy.DirectionFlat, "0.0000", corrID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	fill := fillSub.waitForFill(corrID, 15*time.Second)
	if fill == nil {
		t.Fatal("[S380-DR-2] fill NOT received for flat direction")
	}

	// Flat direction → SideNone → DryRunSubmitter returns StatusAccepted with no fills.
	if !strings.HasPrefix(fill.VenueOrderID, "dryrun-") {
		t.Fatalf("[S380-DR-2] expected dryrun- prefix, got %s", fill.VenueOrderID)
	}
	if fill.ExecutionIntent.Side != domainexec.SideNone {
		t.Fatalf("[S380-DR-2] expected side=none for flat, got %s", fill.ExecutionIntent.Side)
	}
	if fill.Metadata.CorrelationID != corrID {
		t.Fatalf("[S380-DR-2] correlation broken")
	}

	t.Logf("[S380-DR-2] venue_order_id=%s side=%s (flat/no-action confirmed)", fill.VenueOrderID, fill.ExecutionIntent.Side)
	t.Log("[S380-DR-2] PASS — flat direction → DryRunSubmitter no-action receipt")
}

// ---------- S380-DR-3: DryRun + Control Gate Interaction ----------

func TestS380_LiveListenDryRun_ControlGateStillBlocks(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s380-dr3-adapter")
	strategyTracker := healthz.NewTracker("s380-dr3-strategy")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s380-dr3-consumer"),
		"strategy-consumer": strategyTracker,
	}

	// Start halted — DryRunSubmitter is active but gate blocks BEFORE it.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s380-dr3-halted", "s380-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s380-dr3-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s380-test",
		})
	}()

	s380SpawnDryRunSupervisor(t, url, trackers)

	stratPub := natsstrategy.NewPublisher(url, "s380-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer stratPub.Close()

	corrID := fmt.Sprintf("s380-dr3-%d", time.Now().UnixNano())
	event := s380BuildStrategyEvent(t, strategy.DirectionLong, "0.8500", corrID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := stratPub.PublishStrategy(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Gate should block — even with DryRunSubmitter, safety gates come first.
	s341WaitCounter(t, adapterTracker, "processed", 1, 15*time.Second)

	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatal("[S380-DR-3] expected skipped_halt >= 1 when gate halted")
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatalf("[S380-DR-3] expected filled=0 when halted, got %d", adapterTracker.Counter("filled").Load())
	}
	// DryRunSubmitter counters should NOT be incremented since gate blocked before reaching it.
	if adapterTracker.Counter("dryrun_intercepted").Load() != 0 {
		t.Fatalf("[S380-DR-3] expected dryrun_intercepted=0 when gate halted, got %d",
			adapterTracker.Counter("dryrun_intercepted").Load())
	}

	t.Log("[S380-DR-3] PASS — control gate blocks before DryRunSubmitter (safety gates > dry-run)")
}

// ---------- S380-DR-4: DryRun Unique Order IDs Across Pipeline ----------

func TestS380_LiveListenDryRun_UniqueOrderIDsAcrossPipeline(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s380-dr4-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     adapterTracker,
		"venue-consumer":    healthz.NewTracker("s380-dr4-consumer"),
		"strategy-consumer": healthz.NewTracker("s380-dr4-strategy"),
	}

	controlStore := s341SetGate(t, url, domainexec.GateActive, "s380-dr4-active", "s380-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	s380SpawnDryRunSupervisor(t, url, trackers)

	stratPub := natsstrategy.NewPublisher(url, "s380-derive-binary", natsstrategy.DefaultRegistry())
	if err := stratPub.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer stratPub.Close()

	// Publish 3 events and collect order IDs.
	orderIDs := make(map[string]bool)
	for i := 0; i < 3; i++ {
		corrID := fmt.Sprintf("s380-dr4-%d-%d", i, time.Now().UnixNano())
		event := s380BuildStrategyEvent(t, strategy.DirectionLong, "0.9000", corrID)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := stratPub.PublishStrategy(ctx, event)
		cancel()
		if prob != nil {
			t.Fatalf("publish %d: %s", i, prob.Message)
		}

		fill := fillSub.waitForFill(corrID, 15*time.Second)
		if fill == nil {
			t.Fatalf("[S380-DR-4] fill %d NOT received", i)
		}
		if !strings.HasPrefix(fill.VenueOrderID, "dryrun-") {
			t.Fatalf("[S380-DR-4] fill %d missing dryrun- prefix: %s", i, fill.VenueOrderID)
		}
		if orderIDs[fill.VenueOrderID] {
			t.Fatalf("[S380-DR-4] duplicate order ID: %s", fill.VenueOrderID)
		}
		orderIDs[fill.VenueOrderID] = true
	}

	if len(orderIDs) != 3 {
		t.Fatalf("[S380-DR-4] expected 3 unique order IDs, got %d", len(orderIDs))
	}

	t.Logf("[S380-DR-4] 3 unique dryrun- order IDs confirmed across pipeline")
	t.Log("[S380-DR-4] PASS — DryRunSubmitter generates unique order IDs in multi-binary context")
}

// ---------- S380-DR-5: DryRunSubmitter as VenuePort — Never Delegates ----------

func TestS380_DryRunSubmitter_NeverDelegatesInPipelineContext(t *testing.T) {
	// This test uses a bomb adapter to prove DryRunSubmitter never
	// delegates even when composed identically to production.
	tracker := healthz.NewTracker("s380-dr5-adapter")
	bomb := &bombAdapter{}
	dryRun := appexec.NewDryRunSubmitter(bomb).WithTracker(tracker)

	directions := []struct {
		side domainexec.Side
		name string
	}{
		{domainexec.SideBuy, "buy"},
		{domainexec.SideSell, "sell"},
		{domainexec.SideNone, "none"},
	}

	for _, d := range directions {
		t.Run(d.name, func(t *testing.T) {
			intent := domainexec.ExecutionIntent{
				Type:          "paper_order",
				Source:        "binancef",
				Instrument:    btcUSDTPerpIntegration(t),
				Timeframe:     60,
				Side:          d.side,
				Quantity:      "0.001",
				Status:        domainexec.StatusSubmitted,
				Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.95"},
				CorrelationID: fmt.Sprintf("s380-dr5-%s", d.name),
				CausationID:   "s380-dr5-cause",
				Timestamp:     time.Now().UTC(),
			}

			receipt, prob := dryRun.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("unexpected problem: %s", prob.Message)
			}
			if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
				t.Fatalf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
			}
		})
	}

	if tracker.Counter("dryrun_intercepted").Load() != 3 {
		t.Fatalf("expected 3 interceptions, got %d", tracker.Counter("dryrun_intercepted").Load())
	}
	t.Log("[S380-DR-5] PASS — DryRunSubmitter never delegates for any side (bomb adapter survived)")
}
