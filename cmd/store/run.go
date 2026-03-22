package main

import (
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	storeactor "internal/actors/scopes/store"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "store")
	slog.SetDefault(logger)

	logger.Info("store starting")

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	bootstrap.RunPreflight("store", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
	})

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Build health trackers from the canonical pipeline catalog.
	// PipelineTrackerDefs() derives definitions from declarePipelines(),
	// so adding a new pipeline there automatically registers its trackers.
	trackers, err := buildTrackers(config.Pipeline)
	if err != nil {
		logger.Error("build trackers", "error", err)
		os.Exit(1)
	}

	pid := engine.Spawn(storeactor.NewStoreSupervisor(config, trackers), "store")

	// Collect all trackers for health server.
	allTrackers := make([]*healthz.Tracker, 0, len(trackers))
	for _, t := range trackers {
		allTrackers = append(allTrackers, t)
	}

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		allTrackers,
		healthz.WithRuntime("store"),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}

// buildTrackers creates health trackers for all enabled pipeline families.
// Tracker definitions are derived from the canonical pipeline catalog in
// store_supervisor.go — no separate list to maintain.
func buildTrackers(pipeline settings.PipelineConfig) (map[string]*healthz.Tracker, error) {
	trackers := make(map[string]*healthz.Tracker)
	for _, def := range storeactor.PipelineTrackerDefs() {
		if def.IsEnabled(pipeline) {
			trackers[def.ProjectionName] = healthz.NewTracker(def.ProjectionName)
			trackers[def.ConsumerName] = healthz.NewTracker(def.ConsumerName)
		}
	}
	if len(trackers) == 0 {
		return nil, errNoFamiliesEnabled
	}
	return trackers, nil
}

var errNoFamiliesEnabled = &storeStartupError{"no projection families enabled — check pipeline.families in config"}

type storeStartupError struct{ msg string }

func (e *storeStartupError) Error() string { return e.msg }
