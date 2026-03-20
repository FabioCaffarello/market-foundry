package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	adapterch "internal/adapters/clickhouse"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "writer")
	slog.SetDefault(logger)

	logger.Info("writer starting")

	// ── Phase 0: Validate all writer prerequisites before any I/O ────
	//
	// All config invariants are checked up front so that misconfiguration
	// fails fast with an actionable message, before opening connections.

	if !config.NATS.Enabled {
		logger.Error("writer startup blocked: nats.enabled must be true — writer consumes events from NATS JetStream")
		os.Exit(1)
	}

	if prob := config.ClickHouse.ValidateForWriter(); prob != nil {
		logger.Error("writer startup blocked: clickhouse config validation failed", "error", prob)
		os.Exit(1)
	}

	if prob := config.Pipeline.ValidateForWriter(); prob != nil {
		logger.Error("writer startup blocked: pipeline config validation failed", "error", prob)
		os.Exit(1)
	}

	logger.Info("writer config validated",
		"clickhouse_addr", config.ClickHouse.Addr,
		"clickhouse_database", config.ClickHouse.Database,
		"batch_size", config.ClickHouse.BatchSizeOrDefault(),
		"flush_interval", config.ClickHouse.FlushIntervalOrDefault().String(),
		"max_pending", config.ClickHouse.MaxPendingOrDefault(),
		"max_retries", config.ClickHouse.MaxRetriesOrDefault(),
		"nats_url", config.NATS.URL,
	)

	// ── Phase 1: Open connections ────────────────────────────────────

	chClient, err := adapterch.Open(adapterch.Config{
		Addr:     config.ClickHouse.Addr,
		Database: config.ClickHouse.Database,
		Username: config.ClickHouse.Username,
		Password: config.ClickHouse.Password,
	})
	if err != nil {
		logger.Error("writer startup blocked: clickhouse connection failed", "addr", config.ClickHouse.Addr, "error", err)
		os.Exit(1)
	}
	defer chClient.Close()

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Build health trackers from the pipeline catalog.
	trackers, err := buildTrackers(config.Pipeline)
	if err != nil {
		logger.Error("build trackers", "error", err)
		os.Exit(1)
	}

	pid := engine.Spawn(newWriterSupervisor(config, chClient, trackers), "writer")

	// Collect trackers for health server.
	allTrackers := make([]*healthz.Tracker, 0, len(trackers))
	for _, t := range trackers {
		allTrackers = append(allTrackers, t)
	}

	// Start health server with NATS + ClickHouse readiness checks.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{
			bootstrap.NATSReadinessCheck(config),
			{
				Name: "clickhouse",
				Check: func(ctx context.Context) error {
					return chClient.Ping(ctx)
				},
			},
		},
		allTrackers,
		healthz.WithRuntime("writer"),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}

// buildTrackers creates health trackers for all enabled writer pipeline families.
func buildTrackers(pipeline settings.PipelineConfig) (map[string]*healthz.Tracker, error) {
	trackers := make(map[string]*healthz.Tracker)
	for _, def := range writerTrackerDefs() {
		if def.isEnabled(pipeline) {
			trackers[def.consumerName] = healthz.NewTracker(def.consumerName)
			trackers[def.inserterName] = healthz.NewTracker(def.inserterName)
		}
	}
	if len(trackers) == 0 {
		return nil, &writerStartupError{"no writer families enabled — check pipeline config"}
	}
	return trackers, nil
}

type writerStartupError struct{ msg string }

func (e *writerStartupError) Error() string { return e.msg }
