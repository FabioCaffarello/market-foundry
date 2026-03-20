package main

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapterch "internal/adapters/clickhouse"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

const (
	// maxPipelineRestarts is the maximum number of restart attempts per family
	// before the family is marked as degraded. Budget is per process lifetime.
	maxPipelineRestarts = 5

	// pipelineRestartBackoff is the initial backoff delay before the first
	// restart attempt. Each subsequent attempt doubles the delay.
	pipelineRestartBackoff = 2 * time.Second

	// pipelineRestartBackoffCap is the maximum backoff delay between restart
	// attempts.
	pipelineRestartBackoffCap = 30 * time.Second
)

// pipelineState represents the lifecycle state of a pipeline family.
type pipelineState string

const (
	pipelineActive     pipelineState = "active"
	pipelineRestarting pipelineState = "restarting"
	pipelineDegraded   pipelineState = "degraded"
)

// pipelineFailedMsg is sent by a consumer actor to the supervisor when it
// fails to start. The supervisor schedules a restart with exponential backoff.
type pipelineFailedMsg struct {
	family string
	err    error
}

// restartPipelineMsg is a self-scheduled message that triggers a pipeline
// restart after a backoff delay.
type restartPipelineMsg struct {
	family string
}

// pipelineLifecycle tracks the runtime state of a single writer pipeline family.
type pipelineLifecycle struct {
	state       pipelineState
	restarts    int
	lastError   string
	consumerPID *actor.PID
	inserterPID *actor.PID
	pipeline    writerPipeline
}

// writerSupervisor is the root actor for the writer binary.
// It spawns consumer-inserter pairs for each enabled pipeline family,
// projecting domain events from NATS into ClickHouse tables.
//
// Recovery: when a consumer reports startup failure via pipelineFailedMsg,
// the supervisor retries with exponential backoff (2s, 4s, 8s, 16s, 30s)
// up to maxPipelineRestarts. After exhaustion the family is marked degraded
// and other families continue operating normally.
type writerSupervisor struct {
	cfg      settings.AppConfig
	chClient *adapterch.Client
	logger   *slog.Logger
	trackers map[string]*healthz.Tracker

	engine   *actor.Engine
	pid      *actor.PID
	families map[string]*pipelineLifecycle
}

func newWriterSupervisor(config settings.AppConfig, chClient *adapterch.Client, trackers map[string]*healthz.Tracker) actor.Producer {
	return func() actor.Receiver {
		return &writerSupervisor{
			cfg:      config,
			chClient: chClient,
			logger:   slog.Default().With("actor", "writer-supervisor"),
			trackers: trackers,
			families: make(map[string]*pipelineLifecycle),
		}
	}
}

func (s *writerSupervisor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.engine = c.Engine()
		s.pid = c.PID()
		if err := s.start(c); err != nil {
			s.logger.Error("start writer supervisor", "error", err)
			c.Engine().Poison(c.PID())
		}

	case actor.Stopped:
		s.logger.Info("writer supervisor stopped")

	case pipelineFailedMsg:
		s.handlePipelineFailure(c, msg)

	case restartPipelineMsg:
		s.handlePipelineRestart(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		s.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (s *writerSupervisor) start(ctx *actor.Context) error {
	pipelines := declareWriterPipelines(s.chClient)

	var enabledFamilies []string
	var durables []string

	for _, p := range pipelines {
		if !p.isEnabled(s.cfg.Pipeline) {
			s.logger.Info("writer pipeline skipped", "family", p.family)
			continue
		}

		lc := s.spawnPipeline(ctx, p)
		s.families[p.family] = lc

		enabledFamilies = append(enabledFamilies, p.family)
		durables = append(durables, p.consumerSpec.Durable)
	}

	if len(enabledFamilies) == 0 {
		return fmt.Errorf("no writer pipelines enabled — check pipeline config")
	}

	s.logger.Info("writer supervisor started",
		"pipelines", enabledFamilies,
		"consumers", durables,
		"batch_size", s.cfg.ClickHouse.BatchSizeOrDefault(),
		"flush_interval", s.cfg.ClickHouse.FlushIntervalOrDefault().String(),
		"max_pending", s.cfg.ClickHouse.MaxPendingOrDefault(),
		"max_retries", s.cfg.ClickHouse.MaxRetriesOrDefault(),
		"initial_backoff", s.cfg.ClickHouse.InitialBackoffOrDefault().String(),
	)
	return nil
}

// spawnPipeline creates a consumer-inserter actor pair for the given pipeline.
func (s *writerSupervisor) spawnPipeline(ctx *actor.Context, p writerPipeline) *pipelineLifecycle {
	inserterTracker := s.trackers[p.inserterName]
	consumerTracker := s.trackers[p.consumerName]

	// Spawn inserter first — consumer sends rows to it.
	inserterPID := ctx.SpawnChild(newInserterActor(inserterConfig{
		client:         s.chClient,
		family:         p.family,
		table:          p.table,
		insertSQL:      p.insertSQL,
		batchSize:      s.cfg.ClickHouse.BatchSizeOrDefault(),
		flushInterval:  s.cfg.ClickHouse.FlushIntervalOrDefault(),
		maxPending:     s.cfg.ClickHouse.MaxPendingOrDefault(),
		maxRetries:     s.cfg.ClickHouse.MaxRetriesOrDefault(),
		initialBackoff: s.cfg.ClickHouse.InitialBackoffOrDefault(),
		tracker:        inserterTracker,
	}), p.inserterName)

	// Spawn consumer — wired to send decoded events to the inserter.
	consumerPID := ctx.SpawnChild(newWriterConsumerActor(writerConsumerConfig{
		family:        p.family,
		natsURL:       s.cfg.NATS.URL,
		consumerSpec:  p.consumerSpec,
		inserterPID:   inserterPID,
		tracker:       consumerTracker,
		startConsumer: p.startConsumer,
		supervisorPID: ctx.PID(),
	}), p.consumerName)

	return &pipelineLifecycle{
		state:       pipelineActive,
		pipeline:    p,
		consumerPID: consumerPID,
		inserterPID: inserterPID,
	}
}

// handlePipelineFailure processes a consumer startup failure. It either
// schedules a restart with exponential backoff or marks the family as degraded
// when the restart budget is exhausted.
func (s *writerSupervisor) handlePipelineFailure(c *actor.Context, msg pipelineFailedMsg) {
	lc, ok := s.families[msg.family]
	if !ok {
		s.logger.Error("pipeline failure for unknown family", "family", msg.family)
		return
	}
	if lc.state == pipelineDegraded || lc.state == pipelineRestarting {
		return
	}

	lc.restarts++
	lc.lastError = msg.err.Error()

	if t := s.trackers[lc.pipeline.consumerName]; t != nil {
		t.RecordError()
		t.Counter("pipeline_restarts").Add(1)
	}

	if lc.restarts > maxPipelineRestarts {
		lc.state = pipelineDegraded
		s.logger.Error("pipeline degraded — restart budget exhausted",
			"family", msg.family,
			"restarts", lc.restarts,
			"last_error", lc.lastError,
		)
		if t := s.trackers[lc.pipeline.consumerName]; t != nil {
			t.Counter("pipeline_degraded").Add(1)
		}
		s.poisonPipeline(c, lc)
		return
	}

	backoff := s.calcBackoff(lc.restarts)
	lc.state = pipelineRestarting

	s.logger.Warn("pipeline failure — scheduling restart",
		"family", msg.family,
		"error", msg.err,
		"restart", lc.restarts,
		"max_restarts", maxPipelineRestarts,
		"backoff", backoff.String(),
	)

	s.poisonPipeline(c, lc)

	time.AfterFunc(backoff, func() {
		if s.engine != nil && s.pid != nil {
			s.engine.Send(s.pid, restartPipelineMsg{family: msg.family})
		}
	})
}

// handlePipelineRestart respawns a pipeline family after a backoff delay.
func (s *writerSupervisor) handlePipelineRestart(c *actor.Context, msg restartPipelineMsg) {
	lc, ok := s.families[msg.family]
	if !ok || lc.state == pipelineDegraded {
		return
	}

	s.logger.Info("restarting pipeline",
		"family", msg.family,
		"attempt", lc.restarts,
	)

	newLC := s.spawnPipeline(c, lc.pipeline)
	newLC.restarts = lc.restarts
	newLC.lastError = lc.lastError
	s.families[msg.family] = newLC
}

// poisonPipeline stops both consumer and inserter actors for a pipeline family.
func (s *writerSupervisor) poisonPipeline(c *actor.Context, lc *pipelineLifecycle) {
	if lc.consumerPID != nil {
		c.Engine().Poison(lc.consumerPID)
		lc.consumerPID = nil
	}
	if lc.inserterPID != nil {
		c.Engine().Poison(lc.inserterPID)
		lc.inserterPID = nil
	}
}

// calcBackoff returns the exponential backoff duration for the given restart
// count. Starts at pipelineRestartBackoff (2s) and doubles each time, capped
// at pipelineRestartBackoffCap (30s).
func (s *writerSupervisor) calcBackoff(restart int) time.Duration {
	backoff := pipelineRestartBackoff
	for i := 1; i < restart; i++ {
		backoff *= 2
		if backoff > pipelineRestartBackoffCap {
			return pipelineRestartBackoffCap
		}
	}
	return backoff
}
